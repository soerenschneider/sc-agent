package http_server

import (
	"context"
	"encoding/json"
)

func (s *HttpServer) PowerPostAction(ctx context.Context, request PowerPostActionRequestObject) (PowerPostActionResponseObject, error) {
	if s.services.PowerStatus == nil {
		return PowerPostAction501ApplicationProblemPlusJSONResponse{}, nil
	}

	var err error
	switch request.Params.Action {
	case Reboot:
		err = s.services.PowerStatus.Reboot()
	case Shutdown:
		err = s.services.PowerStatus.Shutdown()
	}

	if err != nil {
		return PowerPostAction500ApplicationProblemPlusJSONResponse{}, nil
	}

	return PowerPostAction200Response{}, nil
}

func (s *HttpServer) PowerRebootManagerGetStatus(ctx context.Context, request PowerRebootManagerGetStatusRequestObject) (PowerRebootManagerGetStatusResponseObject, error) {
	if s.services.PowerStatus == nil {
		return PowerRebootManagerGetStatus501ApplicationProblemPlusJSONResponse{}, nil
	}

	data := s.services.RebootManager.Status()
	// TODO: fix openapi sepc
	marshalled, err := json.Marshal(data)
	if err != nil {
		return PowerRebootManagerGetStatus500ApplicationProblemPlusJSONResponse{}, nil
	}
	marshalledStr := string(marshalled)
	return PowerRebootManagerGetStatus200JSONResponse{&marshalledStr}, nil
}

func (s *HttpServer) PowerRebootManagerPostStatus(ctx context.Context, request PowerRebootManagerPostStatusRequestObject) (PowerRebootManagerPostStatusResponseObject, error) {
	if s.services.PowerStatus == nil {
		return PowerRebootManagerPostStatus500ApplicationProblemPlusJSONResponse{}, nil
	}

	switch request.Params.Action {
	case Pause:
		s.services.RebootManager.Pause()
	case Unpause:
		s.services.RebootManager.Unpause()
	}

	return PowerRebootManagerPostStatus200Response{}, nil
}
