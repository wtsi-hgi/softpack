package backend

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/wtsi-hgi/softpack/build"
	"github.com/wtsi-hgi/softpack/db"
)

// type BuildResponse struct {
// 	artefacts *build.Artefacts
// 	err       error
// buildTime time.Duration
// }

type BuildingEnv struct {
	db.Environment
	start time.Time
}

func (s *Server) Build(env db.Environment) error {
	// ch := make(chan BuildResponse)
	go func() {
		// s.buildingEnvs[env.ID] = BuildingEnv{
		// 	env,
		// 	time.Now(),
		// }

		// TODO: Surely I should be submitting the environment name to the builder?

		_, err := build.Build(
			s.config.BaseImgPath,
			s.config.TempDir,
			s.config.InstallDir,
			s.config.WrapperScript,
			s.config.AptSrc,
			toBuildPkg(env.Packages),
		)
		if err != nil {
			return
		}

		// TODO: Where do I put the artefacts? S3?

		// ch <- BuildResponse{
		// 	artefacts: artefacts,
		// 	err:       err,
		// buildTime: time.Since(start),
		// }
	}()

	return nil
}

func (s *Server) GetAverageBuildTime(w http.ResponseWriter, _ *http.Request) error {
	avrg := 1 * time.Hour // TODO: Calculate based on past build responses

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(avrg); err != nil {
		return err
	}

	return nil
}

func toBuildPkg(pkgs []db.Package) (output []build.Package) {
	for _, pkg := range pkgs {
		output = append(output, build.Package{
			Name:    pkg.Name,
			Version: pkg.Version,
		})
	}

	return output
}
