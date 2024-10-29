package http_server

import (
	"context"
)

func (s *HttpServer) WolPostMessage(_ context.Context, request WolPostMessageRequestObject) (WolPostMessageResponseObject, error) {
	if s.services.Wol == nil {
		return WolPostMessage501ApplicationProblemPlusJSONResponse{
			NotImplementedApplicationProblemPlusJSONResponse{
				Detail:   nil,
				Instance: nil,
				Status:   nil,
				Title:    nil,
				Type:     nil,
			},
		}, nil
	}

	if err := s.services.Wol.WakeUp(request.Alias); err != nil {
		return WolPostMessage500ApplicationProblemPlusJSONResponse{
			InternalServerErrorApplicationProblemPlusJSONResponse{
				Detail:   nil,
				Instance: nil,
				Status:   nil,
				Title:    nil,
				Type:     nil,
			},
		}, err
	}

	return WolPostMessage200Response{}, nil
}
