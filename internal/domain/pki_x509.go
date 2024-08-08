package domain

import "time"

type CertConfig struct {
	Id         string
	Role       string
	CommonName string
	Ttl        string
	AltNames   []string
	IpSans     []string
}

type X509CertInfo struct {
	Issuer         Issuer
	Subject        string
	Serial         string
	EmailAddresses []string
	NotBefore      time.Time
	NotAfter       time.Time
}

type Issuer struct {
	SerialNumber string
	CommonName   string
}
