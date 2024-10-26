package app

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/internal/metrics"
	"github.com/soerenschneider/sc-agent/internal/services/components/conditional_reboot/group"
	"github.com/soerenschneider/sc-agent/internal/services/components/conditional_reboot/uptime"
	"go.uber.org/multierr"
)

// defaultSafeMinimumSystemUptime prevents reboot loops
const defaultSafeMinimumSystemUptime = 4 * time.Hour

type ConditionalReboot struct {
	groups               []*group.Group
	rebootImpl           Reboot
	rebootRequest        chan *group.Group
	ignoreRebootRequests atomic.Bool

	safeMinSystemUptime time.Duration
}

type Reboot interface {
	Reboot() error
}

type ConditionalRebootOpts func(c *ConditionalReboot) error

func NewConditionalReboot(groups []*group.Group, rebootImpl Reboot, rebootReq chan *group.Group, opts ...ConditionalRebootOpts) (*ConditionalReboot, error) {
	if len(groups) == 0 {
		return nil, errors.New("no groups provided")
	}

	if rebootImpl == nil {
		return nil, errors.New("no reboot impl provided")
	}

	if rebootReq == nil {
		return nil, errors.New("no channel provided")
	}

	c := &ConditionalReboot{
		groups:              groups,
		rebootImpl:          rebootImpl,
		rebootRequest:       rebootReq,
		safeMinSystemUptime: defaultSafeMinimumSystemUptime,
	}

	var errs error
	for _, opt := range opts {
		err := opt(c)
		if err != nil {
			errs = multierr.Append(errs, err)
		}
	}

	return c, errs
}

// IsSafeSystemBootUptimeReached returns whether minimum limit of system uptime has been reached or not. This is used
// to prevent reboot loops.
func (app *ConditionalReboot) IsSafeSystemBootUptimeReached() bool {
	systemUptime, err := uptime.Uptime()
	if err != nil {
		log.Error().Err(err).Str("component", "conditional-reboot").Msgf("could not determine system uptime, rebooting anyway: %v", err)
		return true
	}

	return systemUptime >= app.safeMinSystemUptime
}

func (app *ConditionalReboot) Start() error {
	ctx, cancel := context.WithCancel(context.Background())
	for _, group := range app.groups {
		group.Start(ctx)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	for {
		select {
		case <-sig:
			cancel()
			log.Info().Str("component", "conditional-reboot").Msgf("Received signal, cancelling..")
			return nil

		case group := <-app.rebootRequest:
			log.Info().Str("component", "conditional-reboot").Msgf("Reboot request from group '%s'", group.GetName())
			cancel()
			// TODO: Get rid of lazy way, use waitgroups?!
			time.Sleep(5 * time.Second)
			err := app.tryReboot(group)
			if err != nil {
				metrics.RebootErrors.Set(1)
				log.Error().Str("component", "conditional-reboot").Err(err).Msg("Reboot failed")
			}

			return err
		}
	}
}

type ConditionalRebootStatus struct {
	Groups   map[string]GroupsStatus `json:"groups"`
	IsPaused bool                    `json:"is_paused"`
}

type GroupsStatus struct {
	WantsReboot bool                    `json:"wants_reboot"`
	Agents      map[string]AgentsStatus `json:"agents"`
}

type AgentsStatus struct {
	State         string `json:"state"`
	StateDuration string `json:"duration"`
}

func (app *ConditionalReboot) Status() ConditionalRebootStatus {
	ret := ConditionalRebootStatus{
		IsPaused: app.ignoreRebootRequests.Load(),
		Groups:   map[string]GroupsStatus{},
	}

	for _, group := range app.groups {
		_, ok := ret.Groups[group.GetName()]
		if !ok {
			ret.Groups[group.GetName()] = GroupsStatus{
				Agents: map[string]AgentsStatus{},
			}
		}

		for _, agent := range group.Agents() {
			state := agent.GetState()

			ret.Groups[group.GetName()].Agents[agent.CheckerNiceName()] = AgentsStatus{
				State:         string(state.Name()),
				StateDuration: agent.GetStateDuration().String(),
			}
		}
	}

	return ret
}

func (app *ConditionalReboot) Pause() {
	app.ignoreRebootRequests.Store(true)
	metrics.RebootIsPaused.Set(1)
}

func (app *ConditionalReboot) Unpause() {
	app.ignoreRebootRequests.Store(false)
	metrics.RebootIsPaused.Set(0)
}

func (app *ConditionalReboot) IsPaused() bool {
	return app.ignoreRebootRequests.Load()
}

var once sync.Once

func (app *ConditionalReboot) tryReboot(group *group.Group) error {
	if app.ignoreRebootRequests.Load() {
		log.Warn().Str("component", "conditional-reboot").Msg("Ignoring request to reboot as reboot is currently in pause mode")
	}

	if !app.IsSafeSystemBootUptimeReached() {
		once.Do(func() {
			log.Warn().Str("component", "conditional-reboot").Msgf("Refusing to reboot, safe minimum system uptime (%s) not reached yet", defaultSafeMinimumSystemUptime)
		})

		return nil
	}

	log.Info().Str("component", "conditional-reboot").Msg("Trying to reboot...")
	return app.rebootImpl.Reboot()
}
