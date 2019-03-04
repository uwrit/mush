package stream

import (
	"context"
	"testing"

	"github.com/UW-Medicine-Research-IT/mush/note"
)

type MockBatchProvider struct {
	max     int
	current int
}

func (m *MockBatchProvider) Batch(batchSize int) ([]*note.Note, error) {
	notes := []*note.Note{}
	for i := 0; i < batchSize; i++ {
		if m.current >= m.max {
			return notes, nil
		}
		m.current++
		n := note.New(m.current, "")
		notes = append(notes, n)
	}
	return notes, nil
}

func Test_Streamer(t *testing.T) {
	ctx, cf := context.WithCancel(context.Background())
	svc := &MockBatchProvider{15, 0}
	_, feed := NewRunning(ctx, svc, 5, 2)

	nc := make(chan []*note.Note)
	go func() {
		notes := []*note.Note{}
		for {
			select {
			case n, ok := <-feed:
				if !ok {
					nc <- notes
					return
				}
				notes = append(notes, n)
			}
		}
	}()

	notes := <-nc

	if len(notes) != 15 {
		t.Errorf("unexpected array length %v", len(notes))
	}

	cf()
}
