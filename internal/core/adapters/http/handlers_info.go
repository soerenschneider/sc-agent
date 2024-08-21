package http_server

import (
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog/log"
)

func (s *HttpServer) InfoGetComponents(w http.ResponseWriter, r *http.Request) {
	enabledComponents := s.services.EnabledComponents()

	var dto InfoComponents //nolint:gosimple
	dto = InfoComponents{
		EnabledComponents: enabledComponents,
	}

	jsonData, err := json.Marshal(dto)
	if err != nil {
		writeRfc7807Error(w, http.StatusInternalServerError, "", "")
		return
	}

	w.WriteHeader(http.StatusOK)
	_, err = w.Write(jsonData)
	if err != nil {
		log.Error().Err(err).Str("endpoint", "handleSystemdLogs").Msg("error delivering response")
	}
}
