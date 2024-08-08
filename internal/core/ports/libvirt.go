package ports

type Libvirt interface {
	StartDomain(domain string) error
	RebootDomain(domain string) error
	ShutdownDomain(domain string) error
}

type LibvirtRestartDomainRequest struct {
	Domain string `json:"domain" validate:"required"`
}

type LibvirtStartDomainRequest struct {
	Domain string `json:"domain" validate:"required"`
}

type LibvirtStopDomainRequest struct {
	Domain string `json:"domain" validate:"required"`
}
