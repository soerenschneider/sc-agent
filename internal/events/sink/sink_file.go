package sink

import (
	"context"
	"encoding/csv"
	"os"

	"github.com/cenkalti/backoff/v4"
	"github.com/soerenschneider/sc-agent/internal/events"
)

type FileEventSink struct {
	file string
}

func (e *FileEventSink) Accept(ctx context.Context, event events.Event) error {
	f, err := os.OpenFile(e.file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return backoff.Permanent(err)
	}

	defer f.Close()

	w := csv.NewWriter(f)
	if err := w.Write(asColumn(event)); err != nil {
		return err
	}
	w.Flush()

	return nil
}

func asColumn(event events.Event) []string {
	return []string{
		event.Id,
		event.Source,
		event.Type,
		event.Data,
	}
}
