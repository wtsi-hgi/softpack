package apt

import (
	"errors"

	"github.com/wtsi-hgi/softpack/db"
)

var ErrPackageNotFound = errors.New("package matching index not found")

type Package struct {
	Name        string
	Description string
	Versions    []string
}

// TODO: Mock this to return a []Package to backend

type Server struct {
	packages []Package
}

func New() *Server {
	return &Server{
		packages: []Package{
			{
				Name: "pkg1",
				Versions: []string{
					"1",
					"2",
				},
				Description: "desc1",
			},
			{
				Name: "pkg2",
				Versions: []string{
					"2",
					"5",
					"8",
				},
				Description: "desc2",
			},
			{
				Name: "pkg3",
				Versions: []string{
					"3",
				},
				Description: "desc3",
			},
			{
				Name: "pkg4",
				Versions: []string{
					"4",
				},
				Description: "",
			},
		},
	}
}

func (s *Server) GetAllPackages() ([]Package, error) {
	return s.packages, nil
}

// Allow for empty descriptions, although it technically shouldn't occur
func (s *Server) GetRecipeDescription(idx db.PackageIndex) (string, error) {
	for _, pkg := range s.packages {
		if pkg.Name == idx.Name {
			// if idx.Version != "" && slices.Contains(pkg.Versions, idx.Version) {
			return pkg.Description, nil
			// }
		}
	}

	return "", ErrPackageNotFound
}
