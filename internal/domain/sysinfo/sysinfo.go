package sysinfo

import "strings"

type Sysinfo struct {
	OS OS `json:"os"`
}

type OS struct {
	Name         string `json:"name,omitempty"`
	Vendor       string `json:"vendor,omitempty"`
	Version      string `json:"version,omitempty"`
	Release      string `json:"release,omitempty"`
	Architecture string `json:"architecture,omitempty"`
}

func (s *Sysinfo) IsDebian() bool {
	osName := strings.ToLower(s.OS.Name)
	return strings.Contains(osName, "debian") || strings.Contains(osName, "ubuntu") || strings.Contains(osName, "raspbian")
}

func (s *Sysinfo) IsRedHat() bool {
	osName := strings.ToLower(s.OS.Name)
	return strings.Contains(osName, "redhat") || strings.Contains(osName, "rocky") || strings.Contains(osName, "alma")
}
