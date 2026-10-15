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

	// responses use the same DTOs as the http adapter
	verb := strings.ToLower(path.Base(topic))
	switch verb {
	case "upgrade":
		return nil, h.services.Packages.Upgrade()
	case "list":
		installed, err := h.services.Packages.ListInstalled()
		if err != nil {
			return nil, err
		}
		return httpAdapter.PackagesInstalled{Packages: httpAdapter.ConvertPackagesToDto(installed)}, nil
	case "updates":
		updates, err := h.services.Packages.CheckUpdate()
		if err != nil {
			return nil, err
		}
		return httpAdapter.ConvertPackageUpdatesToDto(updates), nil
	}

	return nil, domain.ErrNotImplemented
}
