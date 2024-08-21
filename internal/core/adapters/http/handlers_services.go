package http_server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/internal/core/ports"
	"github.com/soerenschneider/sc-agent/internal/domain"
)

func (s *HttpServer) ServicesUnitLogsGet(w http.ResponseWriter, r *http.Request, unit string) {
	if s.services.Services == nil {
		writeRfc7807Error(w, http.StatusNotImplemented, "Function not implemented", "")
		return
	}

	logsRequest := ports.SystemdLogsRequest{Unit: unit}
	logs, err := s.services.Services.Logs(logsRequest)
	if err != nil {
		if errors.Is(err, domain.ErrServicesNoSuchUnit) {
			writeRfc7807Error(w, http.StatusNotFound, "no such unit", "")
		}
		return
	}

	var dto ServiceLogsData //nolint:gosimple
	dto = ServiceLogsData{
		Data: &ServiceLogs{
			Logs: logs,
		},
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

func (s *HttpServer) ServicesUnitStatusPut(w http.ResponseWriter, r *http.Request, unit string, params ServicesUnitStatusPutParams) {
	if s.services.Services == nil {
		writeRfc7807Error(w, http.StatusNotImplemented, "Function not implemented", "")
		return
	}

	var err error
	if params.Action == Restart {
		err = s.services.Services.Restart(unit)
	} else if params.Action == Start {
		writeRfc7807Error(w, http.StatusNotImplemented, "Function not implemented", "")
		return
	} else if params.Action == Stop {
		writeRfc7807Error(w, http.StatusNotImplemented, "Function not implemented", "")
		return
	}

	if err != nil {
		writeRfc7807Error(w, http.StatusInternalServerError, "", "")
		return
	}

	w.WriteHeader(http.StatusOK)
}
