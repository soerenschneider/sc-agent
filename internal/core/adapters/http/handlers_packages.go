package http_server

import (
	"context"

	"github.com/soerenschneider/sc-agent/internal/domain"
)

func (s *HttpServer) PackagesInstalledGet(_ context.Context, _ PackagesInstalledGetRequestObject) (PackagesInstalledGetResponseObject, error) {
	if s.services.Packages == nil {
		return PackagesInstalledGet501ApplicationProblemPlusJSONResponse{}, nil
	}

	resp, err := s.services.Packages.ListInstalled()
	if err != nil {
		return PackagesInstalledGet500ApplicationProblemPlusJSONResponse{}, nil
	}

	return PackagesInstalledGet200JSONResponse{
		Packages: convertPackages(resp),
	}, nil
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

func (s *HttpServer) PackagesUpdatesGet(_ context.Context, _ PackagesUpdatesGetRequestObject) (PackagesUpdatesGetResponseObject, error) {
	if s.services.Packages == nil {
		return PackagesUpdatesGet501ApplicationProblemPlusJSONResponse{}, nil
	}

	resp, err := s.services.Packages.CheckUpdate()
	if err != nil {
		return PackagesUpdatesGet500ApplicationProblemPlusJSONResponse{}, nil
	}

	return PackagesUpdatesGet200JSONResponse{
		UpdatablePackages: convertPackages(resp.UpdatablePackages),
		UpdatesAvailable:  resp.UpdatesAvailable,
	}, nil
}

func (s *HttpServer) PackagesUpgradeRequestsPost(_ context.Context, _ PackagesUpgradeRequestsPostRequestObject) (PackagesUpgradeRequestsPostResponseObject, error) {
	if s.services.Packages == nil {
		return PackagesUpgradeRequestsPost501ApplicationProblemPlusJSONResponse{}, nil
	}

	err := s.services.Packages.Upgrade()
	if err != nil {
		return PackagesUpgradeRequestsPost500ApplicationProblemPlusJSONResponse{}, nil
	}

	return PackagesUpgradeRequestsPost200Response{}, nil
}
