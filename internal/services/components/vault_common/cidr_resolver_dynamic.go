package vault_common

import (
	"errors"
	"fmt"
	"net"
	"net/url"

	"github.com/rs/zerolog/log"
)

type DynamicCidrResolver struct {
	vaultAddress string
}

func NewDynamicCidrResolver(vaultAddr string) (*DynamicCidrResolver, error) {
	if len(vaultAddr) == 0 {
		return nil, errors.New("empty vaultAddr passed")
	}

	parsedURL, err := url.Parse(vaultAddr)
	if err != nil {
		return nil, err
	}

	if len(parsedURL.Host) == 0 {
		return nil, fmt.Errorf("invalid vaultAddr %q specified", vaultAddr)
	}

	address := parsedURL.Host
	if parsedURL.Port() == "" { // Port() returns an empty string if no port is specified
		switch parsedURL.Scheme {
		case "http":
			address += ":80"
		case "https":
			address += ":443"
		}
	}

	return &DynamicCidrResolver{vaultAddress: address}, nil
}

func (r *DynamicCidrResolver) GetCidr() ([]string, error) {
	conn, err := net.Dial("tcp", r.vaultAddress)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.TCPAddr)

	ipNet := &net.IPNet{
		IP:   localAddr.IP,
		Mask: net.CIDRMask(32, 32),
	}

	cidr := ipNet.String()
	log.Info().Str("component", approleComponentName).Msgf("resolved dynamic cidr list to %q", cidr)
	return []string{cidr}, nil
}
