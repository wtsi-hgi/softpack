package backend

import (
	"encoding/json"
	"net/http"
	"os/user"
)

func getUserGroups(username string) []string {
	u, err := user.Lookup(username)
	if err != nil {
		return nil
	}

	gids, err := u.GroupIds()
	if err != nil {
		return nil
	}

	gs := make([]string, 0, len(gids))

	for _, gid := range gids {
		grp, err := user.LookupGroupId(gid)
		if err != nil {
			return nil
		}

		gs = append(gs, grp.Name)
	}

	return gs
}

func (s *Server) GetGroups(w http.ResponseWriter, r *http.Request) error {
	username, err := GetItemFromRequest[string](r)
	if err != nil {
		return err
	}

	groups := getUserGroups(username)

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(groups); err != nil {
		return err
	}

	return nil
}
