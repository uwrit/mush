package note

import "fmt"

// Note ...
type Note struct {
	ID   int
	Text string `json:"Text"`
}

func (n *Note) String() string {
	return fmt.Sprintf("{\"Id\":%d}", n.ID)
}

// New ...
func New(id int, text string) *Note {
	return &Note{
		ID:   id,
		Text: text,
	}
}

// Status represents the status of a note result.
type Status int

// Result ...
type Result struct {
	ID     int    `json:"Id"`
	Status Status `json:"Status"`
	Err    error  `json:"Error"`
	Body   string `json:"Body"`
}

func (r *Result) String() string {
	return fmt.Sprintf("{\"Id\":%d,\"Status\":%d,\"Err\":\"%s\",\"Body\":\"%s\"}", r.ID, r.Status, r.Err, r.Body)
}
