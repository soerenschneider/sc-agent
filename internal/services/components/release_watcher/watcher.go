package release_watcher

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/internal/metrics"
)

const (
	componentName = "release_watcher"
	owner         = "soerenschneider"
	repo          = "sc-agent"
)

type Client interface {
	Do(r *http.Request) (*http.Response, error)
}

type ReleaseWatcher struct {
	client     Client
	once       sync.Once
	tag        string
	updateSeen bool
}

type GitHubRelease struct {
	TagName string `json:"tag_name"`
}

func New(client Client, tag string) (*ReleaseWatcher, error) {
	if client == nil {
		return nil, errors.New("nil client passed")
	}
	if len(tag) == 0 {
		return nil, errors.New("empty release tag passed")
	}
	return &ReleaseWatcher{
		client: client,
		tag:    tag,
	}, nil
}

func (r *ReleaseWatcher) WatchReleases(ctx context.Context) {
	r.once.Do(func() {
		ticker := time.NewTicker(12 * time.Hour)
		// we don't want to wait one tick for getting the data, however, if we're in a restart loop, we don't want
		// to hammer GitHub API and block our IP. 30s should be more than enough to be safe from a restart loop.
		timer := time.NewTimer(30 * time.Second)

		for {
			select {
			case <-timer.C:
				r.CheckRelease(ctx)
			case <-ticker.C:
				r.CheckRelease(ctx)
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	})
}

func (r *ReleaseWatcher) CheckRelease(ctx context.Context) {
	release, err := GetLatestRelease(ctx, r.client, owner, repo)
	if err != nil {
		metrics.UpdateCheckErrors.Inc()
		log.Error().Str("component", componentName).Err(err).Msg("check for latest release failed")
		return
	}

	if release != r.tag {
		if !r.updateSeen {
			log.Info().Str("component", componentName).Str("remote_version", release).Str("local_version", r.tag).Msg("noticed update")
			metrics.UpdateAvailable.Set(1)
		}
		r.updateSeen = true
	} else {
		log.Debug().Str("component", componentName).Str("remote_version", release).Str("local_version", r.tag).Msg("no update available")
		metrics.UpdateAvailable.Set(0)
	}
}

// GetLatestRelease fetches the latest release tag from a GitHub repository
func GetLatestRelease(ctx context.Context, client Client, owner, repo string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch latest release: %s", resp.Status)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}

	return release.TagName, nil
}
