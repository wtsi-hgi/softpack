package apt

import (
	"bufio"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"pault.ag/go/debian/control"
)

func readIndex(url string) ([]Package, error) {
	if strings.HasPrefix(url, "s3://") {
		return readS3Index(url)
	} else if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		return readHTTPIndex(url)
	}

	return readFile(url)
}

func readS3Index(s3URL string) ([]Package, error) {
	u, err := url.Parse(s3URL)
	if err != nil {
		return nil, err
	}

	u.Path = strings.TrimPrefix(u.Path, "/")

	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to load S3 configuration, %w", err)
	}

	client := s3.NewFromConfig(cfg)

	obj, err := client.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: &u.Host,
		Key:    &u.Path,
	})
	if err != nil {
		return nil, err
	}

	return parseIndex(obj.Body, strings.HasSuffix(u.Path, ".gz"))
}

func readHTTPIndex(url string) ([]Package, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	return parseIndex(resp.Body, strings.HasSuffix(url, ".gz"))
}

func readFile(path string) ([]Package, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	return parseIndex(f, strings.HasSuffix(path, ".gz"))
}

func parseIndex(r io.ReadCloser, compressed bool) ([]Package, error) {
	defer r.Close()

	if compressed {
		var err error

		if r, err = gzip.NewReader(r); err != nil {
			return nil, err
		}
	}

	index, err := control.ParseBinaryIndex(bufio.NewReader(r))
	if err != nil {
		return nil, err
	}

	var packages []Package

	for _, entry := range index {
		if _, ok := entry.Values["XB-Softpack"]; !ok {
			continue
		}

		pos, exists := slices.BinarySearchFunc(packages, Package{Name: entry.Package}, func(a, b Package) int {
			return strings.Compare(a.Name, b.Name)
		})
		if !exists {
			packages = slices.Insert(packages, pos, Package{
				Name:        entry.Package,
				Description: entry.Description,
			})
		}

		packages[pos].Versions = append(packages[pos].Versions, entry.Version.Version)
	}

	return packages, nil
}
