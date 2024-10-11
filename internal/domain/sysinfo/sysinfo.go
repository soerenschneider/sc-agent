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
	return strings.Contains(strings.ToLower(s.OS.Name), "debian")
}

func (s *Sysinfo) IsRedHat() bool {
	return strings.Contains(strings.ToLower(s.OS.Name), "redhat")
}
