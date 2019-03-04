package sink

import (
	"context"
	"log"
	"sync"

	"github.com/UW-Medicine-Research-IT/mush/note"
)

// Writer abstracts away the ability to persist a note.Result.
type Writer interface {
	Write(r *note.Result) error
}

// Sink represents a parameterizable pool of concurrent IO sinks for recording results.
type Sink struct {
	concurrency int
	incoming    chan *note.Result
	done        chan struct{}
	ctx         context.Context
	wg          sync.WaitGroup
	writer      Writer
}

// Done exposes the channel on which the sink will notify when it has completed all of it's work.
func (s *Sink) Done() <-chan struct{} { return s.done }

// Accept queues the result for the sink pool.
func (s *Sink) Accept(r *note.Result) { s.incoming <- r }

// Listen subscribes the sink to a feed of results.
// Example:
// ```
// go sink.Listen(results)
// ```
func (s *Sink) Listen(results <-chan *note.Result) {
	for {
		select {
		case r, ok := <-results:
			if !ok {
				close(s.incoming)
				return
			}
			s.Accept(r)
		case <-s.ctx.Done():
			return
		}
	}
}

// Run starts the sink.
// Example:
// ```
// go sinker.Run()
// ```
func (s *Sink) Run() {
	for i := 0; i < s.concurrency; i++ {
		s.wg.Add(1)
		num := i
		go func() {
			log.Println("Sinker", num, "starting up.")
			for {
				select {
				case r, ok := <-s.incoming:
					if !ok {
						log.Println("Sinker", num, "shutting down.")
						s.wg.Done()
						return
					}
					log.Println("Sinker", num, "writing result for note", r.ID)
					err := s.writer.Write(r)
					if err != nil {
						log.Println(err)
					}
				case <-s.ctx.Done():
					log.Println("Sinker", num, "shutting down.")
					s.wg.Done()
					return
				}
			}
		}()
	}
	s.wg.Wait()
	log.Println("Sink shutting down.")
	s.done <- struct{}{}
}

// NewRunning creates a result sink, ready to accept results.
func NewRunning(ctx context.Context, loc int, writer Writer) *Sink {
	s := New(ctx, loc, writer)
	go s.Run()
	return s
}

// New creates a result sink, it must be run to accept results.
func New(ctx context.Context, loc int, writer Writer) *Sink {
	return &Sink{
		concurrency: loc,
		incoming:    make(chan *note.Result),
		done:        make(chan struct{}),
		ctx:         ctx,
		wg:          sync.WaitGroup{},
		writer:      writer,
	}
}
