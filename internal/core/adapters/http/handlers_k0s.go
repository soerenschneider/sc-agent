package http_server

import (
	"context"
)

func (s *HttpServer) K0sPostAction(_ context.Context, request K0sPostActionRequestObject) (K0sPostActionResponseObject, error) {
	if s.services.K0s == nil {
		return K0sPostAction501ApplicationProblemPlusJSONResponse{}, nil
	}

	var err error
	switch request.Params.Action {
	case K0sPostActionParamsActionStart:
		err = s.services.K0s.Start()
	case K0sPostActionParamsActionStop:
		err = s.services.K0s.Stop()
	}

	if err != nil {
		return K0sPostAction500ApplicationProblemPlusJSONResponse{}, nil
	}

	return K0sPostAction200Response{}, nil
}
