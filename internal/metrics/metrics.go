package metrics

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
)

const (
	namespace                        = "sc_agent"
	defaultMetricsDumpFrequency      = 1 * time.Minute
	defaultMetricsHeartbeatFrequency = 1 * time.Minute
	metricsServerComponent           = "metrics"
)

func StartServer(ctx context.Context, addr string, wg *sync.WaitGroup) error {
	defer wg.Done()

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	server := http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadTimeout:       1 * time.Second,
		ReadHeaderTimeout: 1 * time.Second,
		WriteTimeout:      1 * time.Second,
		IdleTimeout:       90 * time.Second,
	}

	errChan := make(chan error)
	go func() {
		log.Info().Str("component", metricsServerComponent).Str("address", addr).Msg("Starting server")
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- fmt.Errorf("can not start metrics server: %w", err)
		}
	}()

	ticker := time.NewTicker(defaultMetricsHeartbeatFrequency)
	for {
		select {
		case <-ticker.C:
			Heartbeat.SetToCurrentTime()
		case <-ctx.Done():
			ticker.Stop()
			log.Info().Str("component", metricsServerComponent).Msg("Stopping server")
			return server.Shutdown(ctx)
		case err := <-errChan:
			ticker.Stop()
			return err
		}
	}
}
