package http_server

import (
	"net/http"
)

func (s *HttpServer) LibvirtPostDomainAction(w http.ResponseWriter, r *http.Request, domain string, action LibvirtPostDomainActionParamsAction) {
	if s.services.Libvirt == nil {
		writeRfc7807Error(w, http.StatusNotImplemented, "Function not implemented", "")
		return
	}

	var err error
	if action == LibvirtPostDomainActionParamsActionReboot {
		err = s.services.Libvirt.RebootDomain(domain)
	} else if action == LibvirtPostDomainActionParamsActionShutdown {
		err = s.services.Libvirt.ShutdownDomain(domain)
	} else if action == LibvirtPostDomainActionParamsActionStart {
		err = s.services.Libvirt.StartDomain(domain)
	}

	if err != nil {
		writeRfc7807Error(w, http.StatusInternalServerError, "domain could not be started", "")
		return
	}

	w.WriteHeader(http.StatusOK)
}
