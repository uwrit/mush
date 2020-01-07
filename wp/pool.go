// TODO(cspital) a single handler function may not be sufficient in all cases.
// We should refactor that into an interface for Handle and OnError.

package wp

import (
	"context"
	"sync"

	"github.com/uwrit/mush/note"
)

// Handler represents a function that processes a note into a result.
type Handler func(*note.Note) *note.Result

// Runner represents a function that starts the pool.
type Runner func(p *Pool)

type Config struct{}

// Pool represents a parameterizable goroutine worker pool.
type Pool struct {
	runner   Runner
	handler  Handler
	incoming chan *note.Note
	results  chan *note.Result
	ctx      context.Context
	wg       sync.WaitGroup
}

// Listen subscribes the pool to a feed of notes.
// Example:
// ```
// go pool.Listen(results)
// ```
func (p *Pool) Listen(feed <-chan *note.Note) {
	for {
		select {
		case n, ok := <-feed:
			if !ok {
				close(p.incoming)
				return
			}
			p.Accept(n)
		case <-p.ctx.Done():
			return
		}
	}
}

// Accept queues the note for processing.
func (p *Pool) Accept(n *note.Note) { p.incoming <- n }

// Results exposes the output channel from the pool.
func (p *Pool) Results() <-chan *note.Result { return p.results }

// Run starts the pool.
func (p *Pool) Run() {
	p.runner(p)
}

// NewRunning creates a running worker pool, ready to accept work.
func NewRunning(ctx context.Context, runner Runner, handler Handler) (*Pool, <-chan *note.Result) {
	pool, results := New(ctx, runner, handler)
	go pool.Run()
	return pool, results
}

// New creates a worker pool, it must be Run() to be used.
func New(ctx context.Context, runner Runner, handler Handler) (*Pool, <-chan *note.Result) {
	pool := &Pool{
		runner:   runner,
		handler:  handler,
		incoming: make(chan *note.Note),
		results:  make(chan *note.Result),
		ctx:      ctx,
		wg:       sync.WaitGroup{},
	}

	return pool, pool.Results()
}
