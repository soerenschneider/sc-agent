package http_server

import (
	"net/http"
)

func (s *HttpServer) WolPostMessage(w http.ResponseWriter, r *http.Request, alias string) {
	if s.services.Wol == nil {
		writeRfc7807Error(w, http.StatusNotImplemented, "Function not implemented", "")
		return
	}

	if err := s.services.Wol.WakeUp(alias); err != nil {
		writeRfc7807Error(w, http.StatusInternalServerError, "error sending wake-on-lan packet", "")
		return
	}
	w.WriteHeader(http.StatusOK)
}
