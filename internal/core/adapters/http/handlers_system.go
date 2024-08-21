package http_server

import (
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog/log"
)

func (s *HttpServer) PowerPostAction(w http.ResponseWriter, r *http.Request, params PowerPostActionParams) {
	if s.services.PowerStatus == nil {
		writeRfc7807Error(w, http.StatusNotImplemented, "Function not implemented", "")
		return
	}

	var err error
	if params.Action == Reboot {
		err = s.services.PowerStatus.Reboot()
	} else if params.Action == Shutdown {
		err = s.services.PowerStatus.Shutdown()
	}

	if err != nil {
		writeRfc7807Error(w, http.StatusInternalServerError, "could not shutdown machine", "")
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *HttpServer) PowerConditionalRebootGetStatus(w http.ResponseWriter, r *http.Request) {
	if s.services.ConditionalReboot == nil {
		writeRfc7807Error(w, http.StatusNotImplemented, "Function not implemented", "")
		return
	}

	data := s.services.ConditionalReboot.Status()

	jsonData, err := json.Marshal(data)
	if err != nil {
		writeRfc7807Error(w, http.StatusInternalServerError, "could not shutdown machine", "")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(jsonData)
	if err != nil {
		log.Error().Err(err).Str("endpoint", "handleConditionalReboot").Msg("error delivering response")
	}
}

func (s *HttpServer) PowerConditionalRebootPostStatus(w http.ResponseWriter, r *http.Request, params PowerConditionalRebootPostStatusParams) {
	if s.services.ConditionalReboot == nil {
		writeRfc7807Error(w, http.StatusNotImplemented, "Function not implemented", "")
		return
	}

	if params.Action == Pause {
		s.services.ConditionalReboot.Pause()
	} else if params.Action == Unpause {
		s.services.ConditionalReboot.Unpause()
	}
}
