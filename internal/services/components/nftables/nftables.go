package nftables

import (
	"bytes"
	"fmt"
	"net/netip"
	"os/exec"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/internal/config"
	"github.com/soerenschneider/sc-agent/internal/core/ports"
	"github.com/soerenschneider/sc-agent/internal/domain"
	"github.com/soerenschneider/sc-agent/internal/metrics"
)

const serviceName = "nftables"

type set struct {
	family     string
	table      string
	name       string
	maxTimeout time.Duration
}

type NftablesCmd struct {
	useSudo bool
	sets    map[string]set
}

func New(conf config.Nftables) (*NftablesCmd, error) {
	ret := &NftablesCmd{
		useSudo: conf.UseSudo,
		sets:    make(map[string]set, len(conf.Sets)),
	}

	for alias, s := range conf.Sets {
		var maxTimeout time.Duration
		if s.MaxTimeout != "" {
			var err error
			maxTimeout, err = time.ParseDuration(s.MaxTimeout)
			if err != nil {
				return nil, fmt.Errorf("invalid max_timeout for set %q: %w", alias, err)
			}
		}

		ret.sets[alias] = set{
			family:     s.Family,
			table:      s.Table,
			name:       s.Set,
			maxTimeout: maxTimeout,
		}
	}

	return ret, nil
}

// AddSetElement adds an element to a predefined named set. The element is removed by nftables once the timeout
// expired. If the element is already part of the set, its timeout is reset to the given timeout.
func (n *NftablesCmd) AddSetElement(req ports.NftablesAddSetElementRequest) error {
	s, ok := n.sets[req.Set]
	if !ok {
		metrics.NftablesNoSuchSet.WithLabelValues(req.Set).Inc()
		return domain.ErrNftablesNoSuchSet
	}

	element, err := parseElement(req.Element)
	if err != nil {
		return err
	}

	timeout, err := time.ParseDuration(req.Timeout)
	if err != nil {
		return fmt.Errorf("%w: %q is not a valid duration", domain.ErrNftablesInvalidTimeout, req.Timeout)
	}
	if timeout < time.Second {
		return fmt.Errorf("%w: timeout must be at least 1s", domain.ErrNftablesInvalidTimeout)
	}
	if s.maxTimeout > 0 && timeout > s.maxTimeout {
		return fmt.Errorf("%w: timeout exceeds maximum of %s", domain.ErrNftablesInvalidTimeout, s.maxTimeout)
	}

	// Adding an element that is already part of the set does not update its timeout. Therefore, the element is
	// added (so deleting it can not fail), deleted and added again with the requested timeout in a single atomic
	// transaction.
	setRef := fmt.Sprintf("%s %s %s", s.family, s.table, s.name)
	ruleset := fmt.Sprintf("add element %s { %s }\ndelete element %s { %s }\nadd element %s { %s timeout %ds }\n",
		setRef, element, setRef, element, setRef, element, int64(timeout.Seconds()))

	cmd := []string{"nft", "-f", "-"}

	var c *exec.Cmd
	if n.useSudo {
		c = exec.Command("sudo", cmd...)
	} else {
		c = exec.Command(cmd[0], cmd[1:]...) // #nosec G204
	}

	c.Stdin = strings.NewReader(ruleset)
	stderr := &bytes.Buffer{}
	c.Stderr = stderr

	if err := c.Run(); err != nil {
		metrics.NftablesErrors.WithLabelValues(req.Set).Inc()
		log.Error().Err(err).Str("service", serviceName).Str("set", req.Set).Str("element", element).Str("stderr", strings.TrimSpace(stderr.String())).Msg("could not add element to set")
		return err
	}

	log.Info().Str("service", serviceName).Str("set", req.Set).Str("element", element).Str("timeout", timeout.String()).Msg("added element to set")
	return nil
}

// parseElement only accepts IP addresses and prefixes, as the element is passed verbatim to nft.
func parseElement(element string) (string, error) {
	element = strings.TrimSpace(element)

	if addr, err := netip.ParseAddr(element); err == nil && addr.Zone() == "" {
		return addr.String(), nil
	}

	if prefix, err := netip.ParsePrefix(element); err == nil {
		return prefix.Masked().String(), nil
	}

	return "", fmt.Errorf("%w: %q is neither an ip address nor a prefix", domain.ErrNftablesInvalidElement, element)
}
