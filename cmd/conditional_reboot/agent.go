package deps

import (
	"fmt"

	"github.com/soerenschneider/sc-agent/internal/config"
	"github.com/soerenschneider/sc-agent/internal/services/components/conditional_reboot/agent"
	"github.com/soerenschneider/sc-agent/internal/services/components/conditional_reboot/agent/preconditions"
	"github.com/soerenschneider/sc-agent/internal/services/components/conditional_reboot/checkers"
	"github.com/soerenschneider/sc-agent/internal/sysinfo"
)

func BuildAgent(c *config.AgentConf) (*agent.StatefulAgent, error) {
	checker, err := BuildChecker(c)
	if err != nil {
		return nil, fmt.Errorf("could not build checker: %w", err)
	}

	precondition, err := BuildPrecondition(c)
	if err != nil {
		return nil, fmt.Errorf("could not build precondition: %w", err)
	}

	agent, err := agent.NewAgent(checker, precondition, c)
	if err != nil {
		return nil, err
	}

	return agent, nil
}

func BuildChecker(c *config.AgentConf) (agent.Checker, error) {
	switch c.CheckerName {
	case checkers.RebootCheckerName:
		if sysinfo.Sysinfo.IsDebian() {
			return checkers.NewRebootCheckerApt()
		} else if sysinfo.Sysinfo.IsRedHat() {
			return checkers.NewRebootCheckerDnf()
		}
		return nil, fmt.Errorf("unknown/unsupported system: %v", sysinfo.Sysinfo.OS)
	case checkers.NeedrestartCheckerName:
		return checkers.NeedrestartCheckerFromMap(c.CheckerArgs)
	case checkers.FileCheckerName:
		return checkers.FileCheckerFromMap(c.CheckerArgs)
	case checkers.DnsCheckerName:
		return checkers.DnsCheckerFromMap(c.CheckerArgs)
	case checkers.PrometheusName:
		return checkers.PrometheusCheckerFromMap(c.CheckerArgs)
	case checkers.TcpName:
		return checkers.TcpCheckerFromMap(c.CheckerArgs)
	case checkers.IcmpCheckerName:
		return checkers.IcmpCheckerFromMap(c.CheckerArgs)
	}

	return nil, fmt.Errorf("unknown checker: %s", c.CheckerName)
}

func BuildPrecondition(c *config.AgentConf) (agent.Precondition, error) {
	switch c.PreconditionName {
	case preconditions.WindowedPreconditionName:
		return preconditions.WindowPreconditionFromMap(c.PreconditionArgs)
	case preconditions.AlwaysPreconditionName:
		return &preconditions.AlwaysPrecondition{}, nil
	default:
		return &preconditions.AlwaysPrecondition{}, nil
	}
}
