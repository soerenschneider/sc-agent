package vault_common

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/rs/zerolog/log"
)

const dynamicCidrDialTimeout = 10 * time.Second

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

// GetCidr returns the single-host CIDR of the local address used to connect to Vault. Loopback addresses are rejected:
// the connection may be proxied, in which case Vault sees a different address.
func (r *DynamicCidrResolver) GetCidr() ([]string, error) {
	conn, err := net.DialTimeout("tcp", r.vaultAddress, dynamicCidrDialTimeout)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = conn.Close()
	}()

	localAddr := conn.LocalAddr().(*net.TCPAddr)
	cidr, err := hostCidr(localAddr.IP)
	if err != nil {
		return nil, err
	}

	log.Info().Str("component", approleComponentName).Msgf("resolved dynamic cidr list to %q", cidr)
	return []string{cidr}, nil
}

func hostCidr(ip net.IP) (string, error) {
	if ip.IsLoopback() {
		return "", fmt.Errorf("local address %s is a loopback address, Vault may see a different address if the connection is proxied: use a static cidr resolver instead", ip)
	}

	bits := 8 * net.IPv6len
	if ip4 := ip.To4(); ip4 != nil {
		ip, bits = ip4, 8*net.IPv4len
	}
	ipNet := &net.IPNet{
		IP:   ip,
		Mask: net.CIDRMask(bits, bits),
	}
	return ipNet.String(), nil
}
