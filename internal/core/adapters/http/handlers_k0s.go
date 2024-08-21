package http_server

import (
	"net/http"
)

func (s *HttpServer) K0sPostAction(w http.ResponseWriter, r *http.Request, params K0sPostActionParams) {
	if s.services.K0s == nil {
		writeRfc7807Error(w, http.StatusNotImplemented, "Function not implemented", "")
		return
	}

	var err error
	if params.Action == K0sPostActionParamsActionStart {
		err = s.services.K0s.Start()
	} else if params.Action == K0sPostActionParamsActionStop {
		err = s.services.K0s.Stop()
	}

	if err != nil {
		writeRfc7807Error(w, http.StatusInternalServerError, "could not stop k0s", "")
		return
	}

	w.WriteHeader(http.StatusOK)
}
