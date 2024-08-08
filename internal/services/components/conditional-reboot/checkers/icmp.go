package checkers

import (
	"context"
	"errors"
	"fmt"
	probing "github.com/prometheus-community/pro-bing"
	"runtime"
	"time"
)

const (
	IcmpCheckerName    = "icmp"
	icmpDefaultTimeout = 3 * time.Second
)

type IcmpChecker struct {
	host       string
	timeout    time.Duration
	privileged bool
}

func NewIcmpChecker(host string) (*IcmpChecker, error) {
	if len(host) == 0 {
		return nil, errors.New("empty host provided")
	}

	return &IcmpChecker{
		host:       host,
		timeout:    icmpDefaultTimeout,
		privileged: getPrivilegedDefaultForPlatform(),
	}, nil
}

func getPrivilegedDefaultForPlatform() bool {
	switch runtime.GOOS {
	case "linux":
		return true
	case "windows":
		return true
	}

	return false
}

func IcmpCheckerFromMap(args map[string]any) (*IcmpChecker, error) {
	if args == nil {
		return nil, errors.New("empty args supplied")
	}

	host, ok := args["host"]
	if !ok {
		return nil, errors.New("no 'host' supplied")
	}

	checker, err := NewIcmpChecker(fmt.Sprintf("%s", host))
	if err != nil {
		return nil, err
	}

	timeoutHuman, ok := args["timeout"].(string)
	if ok {
		timeout, err := time.ParseDuration(timeoutHuman)
		if err != nil {
			return nil, fmt.Errorf("timeout duration could not be parsed: %w", err)
		}
		checker.timeout = timeout
	}

	privileged, ok := args["privileged"].(bool)
	if ok {
		checker.privileged = privileged
	}

	return checker, nil
}

func (c *IcmpChecker) Name() string {
	isPrivileged := "privileged"
	if !c.privileged {
		isPrivileged = "un" + isPrivileged
	}

	return fmt.Sprintf("%s://%s (%s)", IcmpCheckerName, c.host, isPrivileged)
}

func (c *IcmpChecker) IsHealthy(ctx context.Context) (bool, error) {
	pinger, err := probing.NewPinger(c.host)
	if err != nil {
		return false, fmt.Errorf("could not create pinger: %w", err)
	}

	count := 1
	pinger.Timeout = 3 * time.Second
	pinger.Count = count
	pinger.SetPrivileged(c.privileged)
	if err := pinger.RunWithContext(ctx); err != nil {
		return false, fmt.Errorf("ping unsuccessful: %w", err)
	}

	stats := pinger.Statistics()
	return stats.PacketsRecv == count, nil
}
