package http_server

import (
	"context"
)

func (s *HttpServer) LibvirtPostDomainAction(_ context.Context, request LibvirtPostDomainActionRequestObject) (LibvirtPostDomainActionResponseObject, error) {
	if s.services.Libvirt == nil {
		return LibvirtPostDomainAction501ApplicationProblemPlusJSONResponse{}, nil
	}

	var err error
	if request.Params.Action == LibvirtPostDomainActionParamsActionReboot {
		err = s.services.Libvirt.RebootDomain(request.Domain)
	} else if request.Params.Action == LibvirtPostDomainActionParamsActionShutdown {
		err = s.services.Libvirt.ShutdownDomain(request.Domain)
	} else if request.Params.Action == LibvirtPostDomainActionParamsActionStart {
		err = s.services.Libvirt.StartDomain(request.Domain)
	}

	if err != nil {
		return LibvirtPostDomainAction500ApplicationProblemPlusJSONResponse{}, nil
	}

	return LibvirtPostDomainAction200Response{}, nil
}
