package cron_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"testing/synctest"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/mmfshirokan/task-manager/internal/cron"
	"github.com/mmfshirokan/task-manager/internal/cron/mocks"
)

func TestOverdueJob(t *testing.T) {
	const interval = time.Minute

	tests := []struct {
		name      string
		runFor    time.Duration
		markErr   error
		wantCalls int
	}{
		{name: "runs immediately on start", runFor: interval / 2, wantCalls: 1},
		{name: "runs on every interval", runFor: 3*interval + interval/2, wantCalls: 4},
		{name: "keeps running after errors", runFor: 2*interval + interval/2, markErr: errors.New("db is down"), wantCalls: 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				marker := mocks.NewMockOverdueMarker(gomock.NewController(t))
				marker.EXPECT().MarkOverdue(gomock.Any()).Return(int64(1), tt.markErr).Times(tt.wantCalls)

				ctx, cancel := context.WithCancel(t.Context())
				done := make(chan struct{})

				go func() {
					cron.New(marker, interval, slog.New(slog.DiscardHandler)).Run(ctx)
					close(done)
				}()

				time.Sleep(tt.runFor)
				synctest.Wait()
				cancel()
				<-done
			})
		})
	}
}
