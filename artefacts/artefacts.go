package artefacts

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Opener interface {
	Open() (io.ReadSeekCloser, error)
}

func Store(url, envPath string, artefacts map[string]Opener) error {
	if strings.HasPrefix(url, "s3://") {
		return storeInS3(url, envPath, artefacts)
	} else if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		return storeInHTTP(url, envPath, artefacts)
	}

	return storeInFS(url, envPath, artefacts)
}

func storeInS3(s3URL, envPath string, artefacts map[string]Opener) error {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return fmt.Errorf("failed to load S3 configuration, %w", err)
	}

	u, err := url.Parse(s3URL)
	if err != nil {
		return fmt.Errorf("failed to parse S3 URL %s: %w", s3URL, err)
	}

	client := s3.NewFromConfig(cfg)

	for name, artefact := range artefacts {
		r, err := artefact.Open()
		if err != nil {
			return fmt.Errorf("failed to open artefact %s: %w", name, err)
		}
		defer r.Close()

		p := path.Join(u.Path, envPath, name)

		if _, err := client.PutObject(context.Background(), &s3.PutObjectInput{
			Bucket: &u.Host, Key: &p, Body: r,
		}); err != nil {
			return fmt.Errorf("failed to upload artefact %s: %w", name, err)
		}
	}

	return nil
}

type HTTPError struct {
	StatusCode int
	Body       string
}

func (h HTTPError) Error() string {
	return fmt.Sprintf("received unexpected status code: %d", h.StatusCode)
}

func storeInHTTP(httpURL, envPath string, artefacts map[string]Opener) error {
	u, err := url.Parse(httpURL)
	if err != nil {
		return fmt.Errorf("failed to parse HTTP URL %s: %w", httpURL, err)
	}

	upath := u.Path

	for name, artefact := range artefacts {
		u.Path = path.Join(upath, envPath, name)

		r, err := artefact.Open()
		if err != nil {
			return fmt.Errorf("failed to open artefact %s: %w", name, err)
		}

		resp, err := http.DefaultClient.Do(&http.Request{Method: http.MethodPut, URL: u, Body: r})
		if err != nil {
			return fmt.Errorf("failed to upload artefact %s: %w", name, err)
		}

		if resp.StatusCode < http.StatusOK || resp.StatusCode > http.StatusResetContent {
			var sb strings.Builder

			io.Copy(&sb, resp.Body) //nolint:errcheck

			return HTTPError{StatusCode: resp.StatusCode, Body: sb.String()}
		}
	}

	return nil
}

func storeInFS(path, envPath string, artefacts map[string]Opener) error {
	for name, artefact := range artefacts {
		ap := filepath.Join(path, envPath, name)

		if err := os.MkdirAll(filepath.Dir(ap), 0755); err != nil { //nolint:mnd
			return err
		}

		r, err := artefact.Open()
		if err != nil {
			return fmt.Errorf("failed to open artefact %s: %w", name, err)
		}
		defer r.Close()

		f, err := os.Create(ap)
		if err != nil {
			return fmt.Errorf("failed to create artefact %s: %w", name, err)
		}

		_, err = io.Copy(f, r)
		if err != nil {
			return fmt.Errorf("failed to write artefact %s: %w", name, err)
		}

		if err = f.Close(); err != nil {
			return fmt.Errorf("failed to close artefact %s: %w", name, err)
		}
	}

	return nil
}

type File string

func (f File) Open() (io.ReadSeekCloser, error) {
	return os.Open(string(f))
}

type Data string

func (d Data) Open() (io.ReadSeekCloser, error) {
	return data{strings.NewReader(string(d))}, nil
}

type data struct {
	*strings.Reader
}

func (data) Close() error {
	return nil
}
