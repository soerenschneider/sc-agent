package http_server

import (
	"net/http"
)

func (s *HttpServer) LibvirtPostDomainAction(w http.ResponseWriter, r *http.Request, domain string, params LibvirtPostDomainActionParams) {
	if s.services.Libvirt == nil {
		writeRfc7807Error(w, http.StatusNotImplemented, "Function not implemented", "")
		return
	}

	var err error
	if params.Action == LibvirtPostDomainActionParamsActionReboot {
		err = s.services.Libvirt.RebootDomain(domain)
	} else if params.Action == LibvirtPostDomainActionParamsActionShutdown {
		err = s.services.Libvirt.ShutdownDomain(domain)
	} else if params.Action == LibvirtPostDomainActionParamsActionStart {
		err = s.services.Libvirt.StartDomain(domain)
	}

	if err != nil {
		writeRfc7807Error(w, http.StatusInternalServerError, "action unsuccessful", "")
		return
	}

	w.WriteHeader(http.StatusOK)
}
