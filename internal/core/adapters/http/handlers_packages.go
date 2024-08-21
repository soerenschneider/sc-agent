package http_server

import (
	"encoding/json"
	"net/http"

	"github.com/soerenschneider/sc-agent/internal/domain"
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

	var dto *PackagesInstalled //nolint:gosimple
	dto = &PackagesInstalled{
		Packages: convertPackages(resp),
	}

	marshalled, _ := json.Marshal(dto)
	_, _ = w.Write(marshalled)
}

func convertPackages(packages []domain.PackageInfo) []PackageInfo {
	ret := make([]PackageInfo, len(packages))
	for index := range packages {
		ret[index] = PackageInfo{
			Name:    packages[index].Name,
			Repo:    packages[index].Repo,
			Version: packages[index].Version,
		}
	}
	return ret
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

	var dto PackageUpdates //nolint:gosimple
	dto = PackageUpdates{
		UpdatablePackages: convertPackages(resp.UpdatablePackages),
		UpdatesAvailable:  resp.UpdatesAvailable,
	}

	marshalled, _ := json.Marshal(dto)
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
