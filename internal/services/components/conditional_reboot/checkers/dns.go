package checkers

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/rs/zerolog/log"
)

const DnsCheckerName = "dns"

type DnsChecker struct {
	host string
}

func NewDnsChecker(host string) (*DnsChecker, error) {
	if len(host) == 0 {
		return nil, errors.New("empty host provided")
	}

	return &DnsChecker{host: host}, nil
}

func DnsCheckerFromMap(args map[string]any) (*DnsChecker, error) {
	if args == nil {
		return nil, errors.New("empty args supplied")
	}

	host, ok := args["host"]
	if !ok {
		return nil, errors.New("no 'host' supplied")
	}

	return NewDnsChecker(fmt.Sprintf("%s", host))
}

func (c *DnsChecker) Name() string {
	return fmt.Sprintf("%s://%s", DnsCheckerName, c.host)
}

func (c *DnsChecker) IsHealthy(_ context.Context) (bool, error) {
	ips, err := net.LookupIP(c.host)
	if err != nil {
		log.Error().Err(err).Str("checker", "dns").Msgf("Connectivity checker '%s' reported error", err)
		return false, nil
	}

	log.Debug().Str("checker", "dns").Msgf("Received reply for checker '%s': %v", c.Name(), ips)
	return true, nil
}
