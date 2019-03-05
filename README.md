# mush

Mush is a pipeline driver and application framework for developing highly concurrent and performant clinical note processing pipelines. It acts as a root event loop for actuating a processing pipeline that may or may not rely on networked services.

It is used internally in UW Medicine Research IT's NLP infrastructure. It can be used to process a static set of notes or listen for notes as they become available.

## Architecture
Mush composes three core pieces of functionality:
- The `stream` package implements a streaming mechanism on top of any static store of clinical notes.
- The `wp` package implements a configurable worker pool for concurrently processing notes into results.
- The `sink` package implements a configurable concurrent IO sink for capturing results.

These packages are connected and managed by the root mush package via the Compose function.

## Benchmarks
To date, we've seen mush sustain ~600,000 notes/hour, CPU 10%, <70MB memory. This run was pointed at an API endpoint that de-identified the notes, the endpoint was capped at 80 TPS. The application was sharing a VM with the SQL Server instance it was using for storage. SQL Server used 90% CPU, 12GB memory.

## Examples
A prototypical mush application must define how to fetch a batch of notes, what to do with a single note, and how to store the results of a single note. Concretely, you must implement a `stream.BatchProvider`, a `wp.Handler`, and a `sink.Writer`.

#### `stream.BatchProvider`
nio/reader.go
```
package nio

import (
    "context"
    "database/sql"

    // driver
    _ "github.com/denisenkom/go-mssqldb"
    "github.com/pkg/errors"
    "github.com/UW-Medicine-Research-IT/mush/note"
)

// NewBatchProvider returns a new stream.BatchProvider for MS SQL Server.
func NewBatchProvider(ctx context.Context, db *sql.DB) *MSSQLBatchProvider {
    return &MSSQLBatchProvider{ctx, db}
}

// MSSQLBatchProvider implements stream.BatchProvider for MS SQL Server.
type MSSQLBatchProvider struct {
    ctx context.Context
    db  *sql.DB
}

// Batch retrieves a batch of notes no greater than provided batchSize.
func (m *MSSQLBatchProvider) Batch(batchSize int) ([]*note.Note, error) {
    tx, err := m.db.BeginTx(m.ctx, nil)
    if err != nil {
        return nil, errors.Wrap(err, "could not start transaction")
    }
    rows, err := tx.QueryContext(m.ctx, "exec dbo.sp_FetchNotes @p1", batchSize)
    if err != nil {
        return nil, errors.Wrap(err, "could not fetch note batch")
    }
    notes := []*note.Note{}
    for rows.Next() {
        var id int
        var text string
        err = rows.Scan(&id, &text)
        if err != nil {
            tx.Rollback()
            return nil, errors.Wrap(err, "could not scan note row")
        }
        notes = append(notes, note.New(id, text))
    }
    return notes, tx.Commit()
}
```

#### `sink.Writer`
nio/writer.go
```
package nio

import (
    "context"
    "database/sql"

    // driver
    _ "github.com/denisenkom/go-mssqldb"
    "github.com/pkg/errors"
    "github.com/UW-Medicine-Research-IT/mush/note"
)

// StatusCodes ...
const (
    ThrottledErr note.Status = -1
    Success      note.Status = 1
    EncodingErr  note.Status = 2
    TooLongErr   note.Status = 3
    ValidateErr  note.Status = 4
    APIErr       note.Status = 5
    MarshalErr   note.Status = 6
)

// NewWriter returns a new sink.Writer for MS SQL Server.
func NewWriter(ctx context.Context, db *sql.DB) *MSSQLWriter {
    return &MSSQLWriter{ctx, db}
}

// MSSQLWriter implements sink.Writer for MS SQL Server.
type MSSQLWriter struct {
    ctx context.Context
    db  *sql.DB
}

// Write persists a note.Result to a SQL Server.
func (w *MSSQLWriter) Write(r *note.Result) error {
    tx, err := w.db.BeginTx(w.ctx, nil)
    if err != nil {
        return errors.Wrapf(err, "could not start transaction to save result: %s", r)
    }
    _, err = tx.ExecContext(w.ctx, "exec dbo.sp_SaveResult @p1, @p2, @p3", r.ID, r.Status, r.Body)
    if err != nil {
        return errors.Wrapf(err, "could not save result: %s", r)
    }
    return tx.Commit()
}
```

cmd/main.go
```
package main

import (
    "context"
    "database/sql"
    "fmt"
    "log"
    "os"
    "unicode/utf8"

    "github.com/my-user/my-project/nio"

    "github.com/UW-Medicine-Research-IT/mush"
    "github.com/UW-Medicine-Research-IT/mush/note"
    "github.com/UW-Medicine-Research-IT/mush/sink"
    "github.com/UW-Medicine-Research-IT/mush/stream"
    "github.com/UW-Medicine-Research-IT/mush/utf"

    "github.com/pkg/errors"

    _ "github.com/denisenkom/go-mssqldb"
)

const databaseConnectionString = "DEMO_MUSH_DBSTRING"

func init() {
    log.SetOutput(os.Stdout)
    log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
}

func main() {
    log.Println("demonstrating mush...")
    ctx, _ := context.WithCancel(context.Background())
    reader, writer := mustGetServices(ctx)
    config := mush.Config {
        StreamBatchSize: 100,
        StreamWaterline: 40,
        PoolWorkerCount: 20,
        SinkWorkerCount: 4,
    }

    musher := mush.Compose(ctx, reader, handle, writer, config)
    musher.Mush()

    musher.Wait()
    log.Println("done!")
}

func mustGetServices(ctx context.Context) (stream.BatchProvider, sink.Writer) {
    cstring := os.Getenv(databaseConnectionString)
    if cstring == "" {
        log.Fatalln(fmt.Sprintf("no connection string found in env var %s", databaseConnectionString))
    }
    db, err := sql.Open("sqlserver", cstring)
    if err != nil {
        log.Fatalln(fmt.Sprintf("could not open db pool: %s", err))
    }
    return nio.NewBatchProvider(ctx, db), nio.NewWriter(ctx, db)
}

// wp.Handler
func handle(n *note.Note) *note.Result {
    log.Println(fmt.Sprintf("processing note %d", n.ID))
    result := note.Result{
        ID: n.ID,
    }

    set := func(status note.Status, e error) *note.Result {
        result.Status = status
        result.Err = e
        if e != nil {
            log.Println("note", n.ID, "processing failed:", &result)
        }
        return &result
    }

    text := utf.EncodeUTF8(n.Text)

    if !utf8.ValidString(text) {
        return set(nio.EncodingErr, errors.New("note contains invalid utf8 characters"))
    }

    if bl := len([]byte(text)); bl >= 20000 {
        return set(nio.TooLongErr, errors.Errorf("note too long: %d", bl))
    }

    result.Body = "{\"Entities\":[],\"UnmappedAttributes\":[]}"
    return set(nio.Success, nil)
}
```