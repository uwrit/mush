package sink

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/UW-Medicine-Research-IT/mush/note"
)

type MockNoteWriter struct {
	count int32
}

func (w *MockNoteWriter) Write(r *note.Result) error {
	atomic.AddInt32(&w.count, 1)
	return nil
}

func Test_Sink(t *testing.T) {
	ctx, cf := context.WithCancel(context.Background())
	svc := new(MockNoteWriter)

	sink := NewRunning(ctx, 3, svc)
	results := make(chan *note.Result)
	go sink.Listen(results)

	go func() {
		for i := 0; i < 10; i++ {
			result := &note.Result{ID: i + 1, Body: ""}
			results <- result
		}
		close(results)
	}()

	<-sink.Done()

	if svc.count != 10 {
		t.Errorf("unexpected results count: %d", svc.count)
	}

	cf()
}
