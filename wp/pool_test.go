package wp

import (
	"context"
	"testing"
	"time"

	"github.com/uwrit/mush/note"
)

type ConcurrentConfig struct {
	concurrency int
}

func handler(n *note.Note) *note.Result {
	time.Sleep(10 * time.Millisecond)
	return &note.Result{
		ID: n.ID,
	}
}

func concurrentRunner(p *Pool) {
	for i := 0; i < 3; i++ {
		p.wg.Add(1)
		go func() {
			for {
				select {
				case n, ok := <-p.incoming:
					if !ok {
						p.wg.Done()
						return
					}
					p.results <- p.handler(n)
				case <-p.ctx.Done():
					p.wg.Done()
					return
				}
			}
		}()
	}
	p.wg.Wait()
	close(p.results)
}

func Test_Pool_With_Accept(t *testing.T) {
	ctx, cf := context.WithCancel(context.Background())
	pool, results := NewRunning(ctx, concurrentRunner, handler)

	rc := make(chan []*note.Result)
	go func() {
		res := []*note.Result{}
		for i := 0; i < 5; i++ {
			res = append(res, <-results)
		}
		rc <- res
	}()

	pool.Accept(note.New(1, ""))
	pool.Accept(note.New(2, ""))
	pool.Accept(note.New(3, ""))
	pool.Accept(note.New(4, ""))
	pool.Accept(note.New(5, ""))

	all := <-rc

	if len(all) != 5 {
		t.Errorf("unexpected array length %v", len(all))
	}

	cf()
}

func Test_Pool_With_Listen(t *testing.T) {
	ctx, cf := context.WithCancel(context.Background())
	pool, results := NewRunning(ctx, concurrentRunner, handler)

	notes := make(chan *note.Note, 10)
	for i := 0; i < 10; i++ {
		notes <- note.New(i+1, "")
	}
	close(notes)

	go pool.Listen(notes)

	count := 0
loop:
	for {
		select {
		case _, ok := <-results:
			if !ok {
				break loop
			}
			count++
		}
	}

	if count != 10 {
		t.Errorf("unexpected count: %d", count)
	}
	cf()
}
