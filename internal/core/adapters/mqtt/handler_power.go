package mqtt

import (
	"context"
	"path"
	"strings"

	"github.com/soerenschneider/sc-agent/internal/core/ports"
	"github.com/soerenschneider/sc-agent/internal/domain"
)

type PowerStatusHandler struct {
	services *ports.Components
}

func (h *PowerStatusHandler) Handle(_ context.Context, topic string, _ []byte) (any, error) {
	if h.services.PowerStatus == nil {
		return nil, domain.ErrComponentDisabled
	}

	verb := strings.ToLower(path.Base(topic))
	switch verb {
	case "shutdown":
		if err := h.services.PowerStatus.Shutdown(); err != nil {
			return nil, err
		}
	case "reboot":
		if err := h.services.PowerStatus.Reboot(); err != nil {
			return nil, err
		}
	}

	return nil, domain.ErrNotImplemented
}
