package cron

//go:generate go tool mockgen -source=cron.go -destination=mocks/cron_mock.go -package=mocks

import (
	"context"
	"log/slog"
	"time"
)

type OverdueMarker interface {
	MarkOverdue(ctx context.Context) (int64, error)
}

// OverdueJob periodically marks unfinished tasks with a passed deadline as
// overdue.
type OverdueJob struct {
	marker   OverdueMarker
	interval time.Duration
	log      *slog.Logger
}

func New(marker OverdueMarker, interval time.Duration, log *slog.Logger) *OverdueJob {
	return &OverdueJob{marker: marker, interval: interval, log: log}
}

// Run executes the job immediately and then on every interval until ctx is
// cancelled.
func (j *OverdueJob) Run(ctx context.Context) {
	j.log.InfoContext(ctx, "overdue job started", "interval", j.interval.String())
	defer j.log.Info("overdue job stopped")

	j.tick(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-time.Tick(j.interval):
			j.tick(ctx)
		}
	}
}

func (j *OverdueJob) tick(ctx context.Context) {
	count, err := j.marker.MarkOverdue(ctx)
	if err != nil {
		if ctx.Err() == nil {
			j.log.ErrorContext(ctx, "mark overdue tasks", "error", err)
		}

		return
	}

	if count > 0 {
		j.log.InfoContext(ctx, "tasks marked overdue", "count", count)

		return
	}

	j.log.DebugContext(ctx, "no overdue tasks")
}
