package checkers

import (
	"context"
	"errors"
	"fmt"
	"net"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
)

const TcpName = "tcp"

var timeout = time.Second * 3

type TcpChecker struct {
	host string
	port string
}

func NewTcpChecker(host string, port string) (*TcpChecker, error) {
	if len(host) == 0 {
		return nil, errors.New("empty host provided")
	}

	if len(port) == 0 {
		return nil, errors.New("empty port provided")
	}

	return &TcpChecker{
		host: host,
		port: port,
	}, nil
}

func TcpCheckerFromMap(args map[string]any) (*TcpChecker, error) {
	if len(args) == 0 {
		return nil, errors.New("can't build tcpchecker, empty args supplied")
	}

	host, ok := args["host"]
	if !ok {
		return nil, errors.New("can't build tcpchecker, no 'host' supplied")
	}

	port, ok := args["port"]
	if !ok {
		return nil, errors.New("can't build tcpchecker, no 'port' supplied")
	}

	return NewTcpChecker(fmt.Sprintf("%s", host), fmt.Sprintf("%s", port))
}

func (c *TcpChecker) Name() string {
	return fmt.Sprintf("%s://%s:%s", TcpName, c.host, c.port)
}

func (c *TcpChecker) IsHealthy(ctx context.Context) (bool, error) {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(c.host, c.port), timeout)
	if err == nil && conn != nil {
		defer conn.Close()
		log.Debug().Str("checker", "tcp").Msgf("Connecting to %s succeeded", c.Name())
		return true, nil
	}

	if errors.Is(err, syscall.ECONNREFUSED) {
		// receiving this error means the remote system replied
		log.Warn().Str("checker", "tcp").Err(err).Msgf("Review configuration, connection refused to %s", c.Name())
		return true, nil
	}

	log.Error().Str("checker", "tcp").Err(err).Msgf("Connectivity checker '%s' encountered errors", c.Name())
	return false, nil
}
