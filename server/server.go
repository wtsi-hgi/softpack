package server

import (
	"log"
	"net/http"

	"github.com/wtsi-hgi/softpack/backend"
)

func Serve(b *backend.Server) {
	http.HandleFunc("/create-environment", b.CreateEnvironment)
	http.HandleFunc("/get-environment", b.GetEnvironment)
	http.HandleFunc("/delete-environment", b.DeleteEnvironment)

	log.Fatal(http.ListenAndServe(":8080", nil))

	// 	/upload - upload artefacts (only needed for tooling).
	// /request-recipe - frontend request for new package.
	// /requested-recipes - frontend request to list requested recipes.
	// /fulfil-requested-recipe - frontend fulfilment of requested recipe.
	// /remove-requested-recipe - frontend request to remove a requested recipe.
	// /get-recipe-description - frontend request to get package description for a hover popup.
	// /build-status - frontend request for average build times (may not be required).
	// /create-environment frontend request to create a new environment.
	// /get-environments - frontend request to list environments.
	// /delete-environment - tooling request to delete environment.
	// /add-tag - frontend request to add a tag to an environment.
	// /set-hidden - frontend request to toggle hidden status on environment.
	// /upload-module - tooling request to add non-Softpack module.
	// /update-module - tooling request to update non-Softpack module.
	// /package-collection - frontend request to list all available packages and versions.
	// /groups - frontend request to get all groups for a username.

}
