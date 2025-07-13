package mqtt

import (
	"context"
	"path"
	"strings"

	httpAdapter "github.com/soerenschneider/sc-agent/internal/core/adapters/http"
	"github.com/soerenschneider/sc-agent/internal/core/ports"
	"github.com/soerenschneider/sc-agent/internal/domain"
)

type PackagesHandler struct {
	services *ports.Components
}

func (h *PackagesHandler) Handle(_ context.Context, topic string, _ []byte) (any, error) {
	if h.services.Packages == nil {
		return nil, domain.ErrComponentDisabled
	}

	verb := strings.ToLower(path.Base(topic))
	switch verb {
	case "upgrade":
		if err := h.services.Packages.Upgrade(); err != nil {
			return nil, err
		}
	case "list":
		installed, err := h.services.Packages.ListInstalled()
		if err != nil {
			return nil, err
		}
		// borrow code from http adapter to convert to dto
		dto := httpAdapter.ConvertPackagesToDto(installed)
		return dto, nil
	}

	return nil, domain.ErrNotImplemented
}
