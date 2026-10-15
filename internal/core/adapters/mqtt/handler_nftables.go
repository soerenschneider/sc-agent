package mqtt

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strings"

	"github.com/soerenschneider/sc-agent/internal/core/ports"
	"github.com/soerenschneider/sc-agent/internal/domain"
)

type NftablesHandler struct {
	services *ports.Components
}

func (h *NftablesHandler) Handle(_ context.Context, topic string, payload []byte) (any, error) {
	if h.services.Nftables == nil {
		return nil, domain.ErrComponentDisabled
	}

	verb := strings.ToLower(path.Base(topic))
	switch verb {
	case "add":
		var req ports.NftablesAddSetElementRequest
		if err := json.Unmarshal(payload, &req); err != nil {
			return nil, fmt.Errorf("%w: could not parse payload: %v", domain.ErrInvalidInput, err)
		}
		return nil, h.services.Nftables.AddSetElement(req)
	}

	return nil, domain.ErrNotImplemented
}
