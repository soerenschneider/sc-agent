package http_server

import (
	"context"
	"errors"

	"github.com/soerenschneider/sc-agent/internal/core/ports"
	"github.com/soerenschneider/sc-agent/internal/domain"
)

func (s *HttpServer) ServicesUnitLogsGet(ctx context.Context, request ServicesUnitLogsGetRequestObject) (ServicesUnitLogsGetResponseObject, error) {
	if s.services.Services == nil {
		return ServicesUnitLogsGet501ApplicationProblemPlusJSONResponse{}, nil
	}

	logsRequest := ports.SystemdLogsRequest{Unit: request.Unit}
	logs, err := s.services.Services.Logs(logsRequest)
	if err != nil {
		if errors.Is(err, domain.ErrServicesNoSuchUnit) {
			return ServicesUnitLogsGet404ApplicationProblemPlusJSONResponse{}, nil
		}
		return ServicesUnitLogsGet500ApplicationProblemPlusJSONResponse{}, nil
	}

	return ServicesUnitLogsGet200JSONResponse{
		Data: &ServiceLogs{logs},
	}, nil
}

func (s *HttpServer) ServicesUnitStatusPut(ctx context.Context, request ServicesUnitStatusPutRequestObject) (ServicesUnitStatusPutResponseObject, error) {
	if s.services.Services == nil {
		return ServicesUnitStatusPut501ApplicationProblemPlusJSONResponse{}, nil
	}

	var err error
	if request.Params.Action == Restart {
		err = s.services.Services.Restart(request.Unit)
	} else if request.Params.Action == Start {
		// TODO: implement
		return ServicesUnitStatusPut501ApplicationProblemPlusJSONResponse{}, nil
	} else if request.Params.Action == Stop {
		// TODO: implement
		return ServicesUnitStatusPut501ApplicationProblemPlusJSONResponse{}, nil
	}

	if err != nil {
		return ServicesUnitStatusPut500ApplicationProblemPlusJSONResponse{}, nil
	}

	return ServicesUnitStatusPut200Response{}, nil
}
