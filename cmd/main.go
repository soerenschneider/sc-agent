package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/internal"
	"github.com/soerenschneider/sc-agent/internal/config"
	"github.com/soerenschneider/sc-agent/internal/metrics"
	"golang.org/x/term"
)

const (
	defaultConfigFile = "/etc/sc-agent.yaml"
	logComponent      = "component"
	mainComponentName = "main"
	waitgroupTimeout  = 20 * time.Second
)

var (
	flagConfigFile   string
	flagDebug        bool
	flagPrintVersion bool
)

func parseFlags() {
	flag.StringVar(&flagConfigFile, "config", defaultConfigFile, "Path of config file")
	flag.BoolVar(&flagDebug, "debug", false, "Print debug logs")
	flag.BoolVar(&flagPrintVersion, "version", false, "Print version and exit")
	flag.Parse()
}

func main() {
	parseFlags()
	if flagPrintVersion {
		fmt.Printf("%s (%s)\n", internal.BuildVersion, internal.CommitHash)
		os.Exit(0)
	}

	metrics.ProcessStartTime.SetToCurrentTime()
	setupLogLevel(flagDebug)
	conf, err := config.ReadConfig(flagConfigFile)
	if err != nil {
		log.Fatal().Err(err).Msg("could not read config file")
	}

	if err := config.Validate(conf); err != nil {
		log.Fatal().Err(err).Msg("invalid configuration")
	}

	services, err := BuildDeps(*conf)
	if err != nil {
		log.Fatal().Err(err).Msg("could not build services")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wg := &sync.WaitGroup{}

	scAgentFatalErrors := make(chan error, 1)

	log.Info().Str(logComponent, mainComponentName).Msgf("%d active components: %v", len(services.EnabledComponents()), services.EnabledComponents())

	if conf.Http != nil && conf.Http.Enabled {
		apiServer, err := buildApiServer(*conf, services)
		if err != nil {
			log.Fatal().Err(err).Msg("could not build api server")
		}
		wg.Add(1)
		go func() {
			if err := apiServer.StartServer(ctx, wg); err != nil {
				if !errors.Is(err, http.ErrServerClosed) {
					scAgentFatalErrors <- fmt.Errorf("could not start api server: %w", err)
				}
			}
		}()
	}

	if conf.Mqtt != nil && conf.Mqtt.Enabled {
		mqttListener, err := buildMqttClient(conf, services)
		if err != nil {
			log.Fatal().Err(err).Msg("could not build mqtt listener")
		}
		wg.Add(1)

		go func() {
			if err := mqttListener.StartListener(ctx, wg); err != nil {
				scAgentFatalErrors <- fmt.Errorf("could not start mqtt listener: %w", err)
			}
		}()
	}

	if conf.Metrics != nil && conf.Metrics.Enabled {
		metricsServer, err := buildMetricsServer(*conf)
		if err != nil {
			log.Fatal().Err(err).Msg("could not build metrics server")
		}

		wg.Add(1)
		go func() {
			err := metricsServer.StartServer(ctx, wg)
			if !errors.Is(err, http.ErrServerClosed) {
				scAgentFatalErrors <- fmt.Errorf("could not start metrics server: %w", err)
			}
		}()
	}

	services.StartServices(ctx, *conf, scAgentFatalErrors)

	// Handle graceful exit
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	var exitCode int
	select {
	case <-sigc:
		log.Info().Str(logComponent, mainComponentName).Msg("received signal")
		exitCode = 0
	case err := <-scAgentFatalErrors:
		log.Error().Str(logComponent, mainComponentName).Err(err).Msg("got fatal error")
		exitCode = 1
	}

	cancel()

	// wait on all members of the waitgroup but end forcefully after the timeout has passed
	gracefulExitDone := make(chan struct{})

	go func() {
		log.Info().Str(logComponent, mainComponentName).Msg("Waiting for components to shut down gracefully")
		wg.Wait()
		close(gracefulExitDone)
	}()

	select {
	case <-gracefulExitDone:
		log.Info().Str(logComponent, mainComponentName).Msg("All components shut down gracefully within the timeout")
	case <-time.After(waitgroupTimeout):
		log.Error().Str(logComponent, mainComponentName).Msg("Components could not be shutdown within timeout, killing process forcefully")
	}
	os.Exit(exitCode)
}

func setupLogLevel(debug bool) {
	level := zerolog.InfoLevel
	if debug {
		level = zerolog.DebugLevel
	}
	zerolog.SetGlobalLevel(level)
	//#nosec:G115
	if term.IsTerminal(int(os.Stdout.Fd())) {
		log.Logger = log.Output(zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: "15:04:05",
		})
	}
}
