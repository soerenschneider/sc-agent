package packages

import (
	"github.com/soerenschneider/sc-agent/internal/domain"
)

type Apt struct {
}

func NewApt() (*Apt, error) {
	return &Apt{}, nil
}

func (m *Apt) ListInstalled() ([]domain.PackageInfo, error) {
	return nil, domain.ErrNotImplemented
}

func (m *Apt) Upgrade() error {
	return domain.ErrNotImplemented
}

func (m *Apt) CheckUpdate() (domain.CheckUpdateResult, error) {
	return domain.CheckUpdateResult{}, domain.ErrNotImplemented
}
