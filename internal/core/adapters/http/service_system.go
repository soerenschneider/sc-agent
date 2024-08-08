package http_server

import (
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog/log"
)

func (s *HttpServer) SystemPowerPostAction(w http.ResponseWriter, r *http.Request, action SystemPowerPostActionParamsAction) {
	if s.services.Machine == nil {
		writeRfc7807Error(w, http.StatusNotImplemented, "Function not implemented", "")
		return
	}

	var err error
	if action == SystemPowerPostActionParamsActionReboot {
		err = s.services.Machine.Reboot()
	} else if action == SystemPowerPostActionParamsActionShutdown {
		err = s.services.Machine.Shutdown()
	}

	if err != nil {
		writeRfc7807Error(w, http.StatusInternalServerError, "could not shutdown machine", "")
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *HttpServer) SystemPowerConditionalRebootGetStatus(w http.ResponseWriter, r *http.Request) {
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

func (s *HttpServer) SystemPowerConditionalRebootPostStatus(w http.ResponseWriter, r *http.Request, action SystemPowerConditionalRebootPostStatusParamsAction) {
	if s.services.ConditionalReboot == nil {
		writeRfc7807Error(w, http.StatusNotImplemented, "Function not implemented", "")
		return
	}

	if action == Pause {
		s.services.ConditionalReboot.Pause()
	} else if action == Unpause {
		s.services.ConditionalReboot.Unpause()
	}
}
