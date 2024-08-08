package ports

import (
	"github.com/soerenschneider/sc-agent/internal/domain"
)

type SystemPackages interface {
	ListInstalled() ([]domain.PackageInfo, error)
	CheckUpdate() (domain.CheckUpdateResult, error)
	Upgrade() error
}
