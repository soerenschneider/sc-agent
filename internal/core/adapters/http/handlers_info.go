package http_server

import (
	"context"
)

func (s *HttpServer) InfoGetComponents(_ context.Context, request InfoGetComponentsRequestObject) (InfoGetComponentsResponseObject, error) {
	enabledComponents := s.services.EnabledComponents()
	return InfoGetComponents200JSONResponse{
		EnabledComponents: enabledComponents,
	}, nil
}
