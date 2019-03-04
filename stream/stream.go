package stream

import (
	"context"
	"log"

	"github.com/UW-Medicine-Research-IT/mush/note"
)

// BatchProvider abstracts away the ability to retrieve a batch of note.Note.
type BatchProvider interface {
	Batch(size int) ([]*note.Note, error)
}

// Streamer implements a streaming mechanism on top of a non-streaming source.
type Streamer struct {
	ctx       context.Context
	svc       BatchProvider
	batchSize int
	buffer    chan *note.Note
	feed      chan *note.Note
}

// Notes exposes the channel on which to listen for new notes.
func (s *Streamer) Notes() <-chan *note.Note { return s.feed }

// Run starts the stream.
// Example:
// ```
// go streamer.Run()
// ```
func (s *Streamer) Run() {
	for {
		if len(s.buffer) == 0 {
			log.Println("fetching new batch of notes")
			notes, err := s.svc.Batch(s.batchSize)
			if err != nil {
				// XXX(cspital) note a huge fan of this, there should be a better way to report errors without crashing out
				log.Fatalln(err)
			}
			num := len(notes)
			if num == 0 {
				log.Println("no notes returned, shutting down the stream")
				close(s.feed)
				close(s.buffer)
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

// NewRunning returns a streamer ready for use.
func NewRunning(ctx context.Context, svc BatchProvider, batchSize int, waterline int) (*Streamer, <-chan *note.Note) {
	hop, notes := New(ctx, svc, batchSize, waterline)
	go hop.Run()
	return hop, notes
}

// New returns a streamer, must be Run to be used.
func New(ctx context.Context, svc BatchProvider, batchSize int, waterline int) (*Streamer, <-chan *note.Note) {
	hop := &Streamer{
		ctx:       ctx,
		svc:       svc,
		batchSize: batchSize,
		buffer:    make(chan *note.Note, batchSize),
		feed:      make(chan *note.Note, waterline),
	}
	return hop, hop.Notes()
}
