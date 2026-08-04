package build

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go/logging"
)

func startServer(aptSrc string) (net.Listener, error) {
	u, err := url.Parse(aptSrc)
	if err != nil {
		return nil, err
	}

	var h http.Handler

	switch u.Scheme {
	case "s3":
		if h, err = newS3Proxy(u); err != nil {
			return nil, err
		}
	case "http", "https":
		h = httputil.NewSingleHostReverseProxy(u)
	default:
		h = http.FileServer(http.Dir(aptSrc))
	}

	l, err := net.Listen("tcp", "127.0.0.1:0") //nolint:noctx
	if err != nil {
		return nil, err
	}

	go http.Serve(l, h) //nolint:errcheck

	return l, nil
}

type s3Proxy struct {
	host   string
	path   string
	client *s3.Client
}

func newS3Proxy(u *url.URL) (http.Handler, error) {
	u.Path = strings.TrimPrefix(u.Path, "/")

	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to load S3 configuration, %w", err)
	}

	cfg.Logger = &logging.Nop{}

	return &s3Proxy{
		host:   u.Host,
		path:   u.Path,
		client: s3.NewFromConfig(cfg),
	}, nil
}

func (s *s3Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p := strings.TrimPrefix(path.Join(s.path, r.URL.Path), "/")

	obj, err := s.client.GetObject(r.Context(), &s3.GetObjectInput{
		Bucket: &s.host,
		Key:    &p,
	})
	if err != nil {
		if _, ok := errors.AsType[*types.NoSuchKey](err); ok { //nolint:errcheck
			http.NotFound(w, r)

			return
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	defer obj.Body.Close()

	io.Copy(w, obj.Body) //nolint:errcheck
}
