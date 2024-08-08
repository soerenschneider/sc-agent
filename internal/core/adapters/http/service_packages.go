package http_server

import (
	"encoding/json"
	"net/http"
)

func (s *HttpServer) PackagesInstalledGet(w http.ResponseWriter, r *http.Request) {
	if s.services.Packages == nil {
		writeRfc7807Error(w, http.StatusNotImplemented, "Function not implemented", "")
		return
	}

	resp, err := s.services.Packages.ListInstalled()
	if err != nil {
		writeRfc7807Error(w, http.StatusInternalServerError, "dnf list installed packages failed", "")
		return
	}

	marshalled, _ := json.Marshal(resp)
	_, _ = w.Write(marshalled)
}

func (s *HttpServer) PackagesUpdatesGet(w http.ResponseWriter, r *http.Request) {
	if s.services.Packages == nil {
		writeRfc7807Error(w, http.StatusNotImplemented, "Function not implemented", "")
		return
	}

	resp, err := s.services.Packages.CheckUpdate()
	if err != nil {
		writeRfc7807Error(w, http.StatusInternalServerError, "dnf check-update failed", "")
		return
	}

	marshalled, _ := json.Marshal(resp)
	_, _ = w.Write(marshalled)
}

func (s *HttpServer) PackagesUpgradeRequestsPost(w http.ResponseWriter, r *http.Request) {
	if s.services.Packages == nil {
		writeRfc7807Error(w, http.StatusNotImplemented, "Function not implemented", "")
		return
	}

	err := s.services.Packages.Upgrade()
	if err != nil {
		writeRfc7807Error(w, http.StatusInternalServerError, "dnf upgrade failed", "")
		return
	}
}
