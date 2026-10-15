package http_server

import (
	"context"
	"errors"

	"github.com/soerenschneider/sc-agent/internal/core/ports"
	"github.com/soerenschneider/sc-agent/internal/domain"
)

func (s *HttpServer) NftablesPostSetElement(_ context.Context, request NftablesPostSetElementRequestObject) (NftablesPostSetElementResponseObject, error) {
	if s.services.Nftables == nil {
		return NftablesPostSetElement501ApplicationProblemPlusJSONResponse{}, nil
	}

	if request.Body == nil {
		return NftablesPostSetElement400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse{Detail: ptr("missing request body")}}, nil
	}

	req := ports.NftablesAddSetElementRequest{
		Set:     request.Set,
		Element: request.Body.Element,
		Timeout: request.Body.Timeout,
	}

	if err := s.services.Nftables.AddSetElement(req); err != nil {
		switch {
		case errors.Is(err, domain.ErrNotFound):
			return NftablesPostSetElement404ApplicationProblemPlusJSONResponse{NotFoundApplicationProblemPlusJSONResponse{Detail: ptr(err.Error())}}, nil
		case errors.Is(err, domain.ErrInvalidInput):
			return NftablesPostSetElement400ApplicationProblemPlusJSONResponse{BadRequestApplicationProblemPlusJSONResponse{Detail: ptr(err.Error())}}, nil
		default:
			return NftablesPostSetElement500ApplicationProblemPlusJSONResponse{}, nil
		}
	}

	return NftablesPostSetElement200Response{}, nil
}

func ptr[T any](v T) *T {
	return &v
}
