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

// type Package struct {
// 	Name        string
// 	Description string `json:"-"`
// 	Versions    []string
// }

type Server struct {
	mu       sync.RWMutex
	packages []db.Package
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

func (s *Server) GetAllPackages() []db.Package {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.packages
}

// Allow for empty descriptions, although it technically shouldn't occur
func (s *Server) GetRecipeDescription(pkg string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pos, ok := slices.BinarySearchFunc(s.packages, db.Package{Name: pkg}, func(a, b db.Package) int {
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

	for _, pkg := range pkgs {
		if exists := s.CheckPackageExists(pkg); !exists {
			return exists
		}
	}

	return true
}

func (s *Server) CheckPackageExists(pkg db.Package) bool {
	for _, aptpkg := range s.packages {
		if CheckPkgEqual(pkg, aptpkg) {
			return true
		}
	}

	return false
}

func CheckPkgEqual(pkg1, pkg2 db.Package) bool {
	return pkg1.Name == pkg2.Name && slices.Equal(pkg1.Versions, pkg2.Versions) && pkg1.Description == pkg2.Description
}
