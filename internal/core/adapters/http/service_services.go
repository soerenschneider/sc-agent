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
	if s.services.Systemd == nil {
		writeRfc7807Error(w, http.StatusNotImplemented, "Function not implemented", "")
		return
	}

	logsRequest := ports.SystemdLogsRequest{Unit: unit}
	logs, err := s.services.Systemd.Logs(logsRequest)
	if err != nil {
		if errors.Is(err, domain.ErrServicesNoSuchUnit) {
			writeRfc7807Error(w, http.StatusNotFound, "no such unit", "")
		}
		return
	}

	ret := &ServiceLogsData{
		Data: &Logs{
			Logs: logs,
		},
	}

	jsonData, err := json.Marshal(ret)

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

func (s *HttpServer) ServicesUnitStatusPut(w http.ResponseWriter, r *http.Request, unit string, action ServicesUnitStatusPutParamsAction) {
	if s.services.Systemd == nil {
		writeRfc7807Error(w, http.StatusNotImplemented, "Function not implemented", "")
		return
	}

	var err error
	if action == ServicesUnitStatusPutParamsActionReboot {
		err = s.services.Systemd.Restart(unit)
	} else if action == ServicesUnitStatusPutParamsActionShutdown {
		writeRfc7807Error(w, http.StatusNotImplemented, "Function not implemented", "")
		return
	} else if action == ServicesUnitStatusPutParamsActionStart {
		writeRfc7807Error(w, http.StatusNotImplemented, "Function not implemented", "")
		return
	}

	if err != nil {
		writeRfc7807Error(w, http.StatusInternalServerError, "", "")
		return
	}

	w.WriteHeader(http.StatusOK)
}
