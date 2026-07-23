package apt

import (
	"errors"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"
)

var ErrPackageNotFound = errors.New("package matching index not found")

type Package struct {
	Name        string
	Description string `json:"-"`
	Versions    []string
}

type Server struct {
	mu       sync.RWMutex
	packages []Package
}

func New(packagesURL string, updateInterval time.Duration) (*Server, error) {
	pkgs, err := readIndex(packagesURL)
	if err != nil {
		return nil, err
	}

	s := &Server{
		packages: pkgs,
	}

	if updateInterval > 0 {
		go s.update(packagesURL, updateInterval)
	}

	return s, nil
}

func (s *Server) update(packagesURL string, updateInterval time.Duration) {
	for {
		time.Sleep(updateInterval)

		pkgs, err := readIndex(packagesURL)
		if err != nil {
			slog.Error("error updating package list", "err", err)

			continue
		}

		s.mu.Lock()
		s.packages = pkgs
		s.mu.Unlock()
	}
}

func (s *Server) GetAllPackages() []Package {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.packages
}

// Allow for empty descriptions, although it technically shouldn't occur
func (s *Server) GetRecipeDescription(pkg string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pos, ok := slices.BinarySearchFunc(s.packages, Package{Name: pkg}, func(a, b Package) int {
		return strings.Compare(a.Name, b.Name)
	})

	if !ok {
		return "", ErrPackageNotFound
	}

	return s.packages[pos].Description, nil
}
