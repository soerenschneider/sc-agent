package sysinfo

import (
	domain "github.com/soerenschneider/sc-agent/internal/domain/sysinfo"
	"github.com/zcalusic/sysinfo"
)

var Sysinfo domain.Sysinfo

func init() {
	var info sysinfo.SysInfo
	info.GetSysInfo()

	Sysinfo = domain.Sysinfo{
		OS: domain.OS{
			Name:         info.OS.Name,
			Vendor:       info.OS.Vendor,
			Version:      info.OS.Version,
			Release:      info.OS.Release,
			Architecture: info.OS.Architecture,
		}}
}
