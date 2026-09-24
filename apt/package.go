package apt

import (
	"errors"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/wtsi-hgi/softpack/db"
)

var ErrInvalidPackage = errors.New("package matching index not found")

type Package struct {
	Name        string   `json:"name"`
	Description string   `json:"-"`
	Versions    []string `json:"versions"`
}

type Server struct {
	mu       sync.RWMutex
	packages []Package
	aliases  map[string]string
}

func New(packagesURL string, updateInterval time.Duration) (*Server, error) {
	pkgs, aliases, err := readIndex(packagesURL)
	if err != nil {
		return nil, err
	}

	s := &Server{
		packages: pkgs,
		aliases:  aliases,
	}

	if updateInterval > 0 {
		go s.update(packagesURL, updateInterval)
	}

	return s, nil
}

func (s *Server) update(packagesURL string, updateInterval time.Duration) {
	for {
		time.Sleep(updateInterval)

		pkgs, aliases, err := readIndex(packagesURL)
		if err != nil {
			slog.Error("error updating package list", "err", err)

			continue
		}

		s.mu.Lock()
		s.packages = pkgs
		s.aliases = aliases
		s.mu.Unlock()
	}
}

func (s *Server) GetAllPackages() []Package {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.packages
}

func (s *Server) GetRecipeDescription(pkg string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pos, ok := slices.BinarySearchFunc(s.packages, Package{Name: pkg}, func(a, b Package) int {
		return strings.Compare(a.Name, b.Name)
	})

	if !ok {
		return "", ErrInvalidPackage
	}

	return s.packages[pos].Description, nil
}

func (s *Server) CheckPackagesExist(pkgs []db.Package) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, reqPkg := range pkgs {
		reqPkg.Name = strings.TrimPrefix(reqPkg.Name, "*")

		if !s.CheckPackageExists(reqPkg) {
			return false
		}
	}

	return true
}

func (s *Server) Alias(pkg string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if alias, ok := s.aliases[pkg]; ok {
		return alias
	}

	name, ver, hasVer := strings.Cut(pkg, "@")

	if alias, ok := s.aliases[name]; ok {
		if hasVer {
			return alias + "@" + ver
		}

		return alias
	}

	return pkg
}

func (s *Server) CheckPackageExists(reqPkg db.Package) bool {
	s.mu.RLock()
	pkgs := s.packages
	s.mu.RUnlock()

	pos, found := slices.BinarySearchFunc(pkgs, Package{Name: reqPkg.Name}, func(a, b Package) int {
		return strings.Compare(a.Name, b.Name)
	})
	if !found {
		return false
	}

	if reqPkg.Version == "" {
		return true
	}

	_, found = slices.BinarySearch(pkgs[pos].Versions, reqPkg.Version)

	return found
}
