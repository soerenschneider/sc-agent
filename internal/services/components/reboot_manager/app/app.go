package app

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/internal/events"
	"github.com/soerenschneider/sc-agent/internal/metrics"
	"github.com/soerenschneider/sc-agent/internal/services/components/reboot_manager/group"
	"github.com/soerenschneider/sc-agent/internal/services/components/reboot_manager/uptime"
	cloudevents "github.com/soerenschneider/soeren.cloud-events/pkg/sc-agent/reboot"
	"go.uber.org/multierr"
)

// defaultSafeMinimumSystemUptime prevents reboot loops
const defaultSafeMinimumSystemUptime = 4 * time.Hour

type RebootManager struct {
	groups               []*group.Group
	rebootImpl           Reboot
	rebootRequest        chan *group.Group
	ignoreRebootRequests atomic.Bool

	safeMinSystemUptime time.Duration
}

type Reboot interface {
	Reboot() error
}

type RebootManagerOpts func(c *RebootManager) error

func NewRebootManager(groups []*group.Group, rebootImpl Reboot, rebootReq chan *group.Group, opts ...RebootManagerOpts) (*RebootManager, error) {
	if len(groups) == 0 {
		return nil, errors.New("no groups provided")
	}

	if rebootImpl == nil {
		return nil, errors.New("no reboot impl provided")
	}

	if rebootReq == nil {
		return nil, errors.New("no channel provided")
	}

	c := &RebootManager{
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
func (app *RebootManager) IsSafeSystemBootUptimeReached() bool {
	systemUptime, err := uptime.Uptime()
	if err != nil {
		log.Error().Err(err).Str("component", "reboot-manager").Msgf("could not determine system uptime, rebooting anyway: %v", err)
		return true
	}

	return systemUptime >= app.safeMinSystemUptime
}

func (app *RebootManager) Start(ctx context.Context) error {
	for _, group := range app.groups {
		group.Start(ctx)
	}

	for {
		select {
		case <-ctx.Done():
			log.Info().Str("component", "reboot-manager").Msgf("Stopping")
			return nil

		case group := <-app.rebootRequest:
			log.Info().Str("component", "reboot-manager").Msgf("Reboot request from group '%s'", group.GetName())
			time.Sleep(5 * time.Second)
			err := app.tryReboot(group)
			if err != nil {
				metrics.RebootErrors.Set(1)
				log.Error().Str("component", "reboot-manager").Err(err).Msg("Reboot failed")
			}

			return err
		}
	}
}

type RebootManagerStatus struct {
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

func (app *RebootManager) Status() RebootManagerStatus {
	ret := RebootManagerStatus{
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

func (app *RebootManager) Pause() {
	app.ignoreRebootRequests.Store(true)
	metrics.RebootIsPaused.Set(1)
}

func (app *RebootManager) Unpause() {
	app.ignoreRebootRequests.Store(false)
	metrics.RebootIsPaused.Set(0)
}

func (app *RebootManager) IsPaused() bool {
	return app.ignoreRebootRequests.Load()
}

var once sync.Once

func (app *RebootManager) tryReboot(group *group.Group) error {
	if app.ignoreRebootRequests.Load() {
		log.Warn().Str("component", "reboot-manager").Msg("Ignoring request to reboot as reboot is currently in pause mode")
	}

	if !app.IsSafeSystemBootUptimeReached() {
		once.Do(func() {
			log.Warn().Str("component", "reboot-manager").Msgf("Refusing to reboot, safe minimum system uptime (%s) not reached yet", defaultSafeMinimumSystemUptime)
		})

		return nil
	}

	log.Info().Str("component", "reboot-manager").Msg("Trying to reboot...")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := events.Accept(ctx, cloudevents.NewSystemRebootedEvent("source", nil)); err != nil {
		log.Error().Err(err).Msg("could not send event")
	}
	return app.rebootImpl.Reboot()
}
