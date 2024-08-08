package ports

import "context"

type ReleaseWatcher interface {
	// CheckRelease checks for a new release of this software and sets communicates it by setting according metrics.
	CheckRelease(ctx context.Context)
	// WatchReleases continously calls CheckRelease.
	WatchReleases(ctx context.Context)
}
