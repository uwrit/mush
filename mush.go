// Package mush implements a pipeline driver for clinical note processing.
package mush

import (
	"context"

	"github.com/UW-Medicine-Research-IT/mush/sink"
	"github.com/UW-Medicine-Research-IT/mush/stream"
	"github.com/UW-Medicine-Research-IT/mush/wp"
)

// Compose bootstraps a Musher from required service implementations and a configuration.
func Compose(ctx context.Context, bp stream.BatchProvider, handler wp.Handler, writer sink.Writer, config Config) Musher {
	streamer, _ := stream.New(ctx, bp, config.StreamBatchSize, config.StreamWaterline)
	pool, _ := wp.New(ctx, config.PoolWorkerCount, handler)
	sinker := sink.New(ctx, config.SinkWorkerCount, writer)

	return &musher{
		ctx:      ctx,
		streamer: streamer,
		pool:     pool,
		sinker:   sinker,
	}
}

// Musher exposes the lifecycle hooks of the mush framework.
type Musher interface {
	Mush()
	Wait()
}

// Config encapsulates configuration data needed for bootstrapping a musher.
type Config struct {
	StreamBatchSize int
	StreamWaterline int

	PoolWorkerCount int

	SinkWorkerCount int
}

type musher struct {
	ctx context.Context

	streamer *stream.Streamer
	pool     *wp.Pool
	sinker   *sink.Sink
}

// Mush starts `1 + (Config.PoolWorkerCount + 2) + (Config.SinkWorkerCount + 2)` goroutines to run the driver.
func (m *musher) Mush() {
	// spin up component event loops
	go m.streamer.Run()
	go m.pool.Run()
	go m.sinker.Run()

	// hook up components to let the work start flowing
	go m.pool.Listen(m.streamer.Notes())
	go m.sinker.Listen(m.pool.Results())
}

// Wait blocks until the sink.Sink has written it's last record.
func (m *musher) Wait() {
	<-m.sinker.Done()
}
