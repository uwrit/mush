package stream

import (
	"context"
	"log"

	"github.com/uwrit/mush/note"
)

// BatchProvider abstracts away the ability to retrieve a batch of note.Note.
type BatchProvider interface {
	Batch(size int) ([]*note.Note, error)
}

// Streamer implements a streaming mechanism on top of a non-streaming source.
type Stream struct {
	ctx       context.Context
	svc       BatchProvider
	batchSize int
	buffer    chan *note.Note
	feed      chan *note.Note
}

// Notes exposes the channel on which to listen for new notes.
func (s *Stream) Notes() <-chan *note.Note { return s.feed }

// Run starts the stream.
// Example:
// ```
// go streamer.Run()
// ```
func (s *Stream) Run() {
	for {
		if len(s.buffer) == 0 {
			log.Println("fetching new batch of notes")
			notes, err := s.svc.Batch(s.batchSize)
			if err != nil {
				log.Println("fetching batch failed, shutting down the stream:", err)
				s.close()
				return
			}
			num := len(notes)
			if num == 0 {
				log.Println("no notes returned, shutting down the stream")
				s.close()
				return
			}
			log.Println("filling buffer with", num, "notes")
			for _, next := range notes {
				s.buffer <- next
			}
		}
		select {
		case next := <-s.buffer:
			s.feed <- next
		case <-s.ctx.Done():
			log.Println("stream shutting down")
			return
		}
	}
}

func (s *Stream) close() {
	close(s.feed)
	close(s.buffer)
}

// NewRunning returns a streamer ready for use.
func NewRunning(ctx context.Context, svc BatchProvider, batchSize int, waterline int) (*Stream, <-chan *note.Note) {
	hop, notes := New(ctx, svc, batchSize, waterline)
	go hop.Run()
	return hop, notes
}

// New returns a streamer, must be Run to be used.
func New(ctx context.Context, svc BatchProvider, batchSize int, waterline int) (*Stream, <-chan *note.Note) {
	hop := &Stream{
		ctx:       ctx,
		svc:       svc,
		batchSize: batchSize,
		buffer:    make(chan *note.Note, batchSize),
		feed:      make(chan *note.Note, waterline),
	}
	return hop, hop.Notes()
}
