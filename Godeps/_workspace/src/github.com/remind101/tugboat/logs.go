package tugboat

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"time"
)

// LogLine represents a line of log output.
type LogLine struct {
	// A unique identifier for this log line.
	ID string `db:"id"`

	// The associated deployment.
	DeploymentID string `db:"deployment_id"`

	// The line of text from the log line.
	Text string `db:"text"`

	// The time that the line was recorded.
	At time.Time `db:"at"`
}

// LogLinesCreate inserts a LogLine into the store.
func (s *store) LogLinesCreate(l *LogLine) error {
	return s.db.Insert(l)
}

// LogLines returns a slice of all LogLines for a given Deployment.
func (s *store) LogLines(d *Deployment) ([]*LogLine, error) {
	var lines []*LogLine
	_, err := s.db.Select(&lines, `select * from logs where deployment_id = $1 order by at asc`, d.ID)
	return lines, err
}

// logsService wraps the LogLinesCreate method.
type logsService interface {
	LogLinesCreate(*LogLine) error
}

// newLogsService returns a new composed logsService.
func newLogsService(store *store, pusher Pusher) logsService {
	return &pushedLogsService{
		logsService: store,
		pusher:      pusher,
	}
}

// pushedLogsService wraps a logsService to send events to pusher.
type pushedLogsService struct {
	logsService

	pusher Pusher
}

// LogLinesCreate sends a pusher event including the new LogLine then delegates
// to the wrapped logsService.
func (s *pushedLogsService) LogLinesCreate(l *LogLine) error {
	channel := deploymentChannel(l.DeploymentID)

	data := struct {
		ID     string `json:"id"`
		Output string `json:"output"`
	}{
		ID:     l.DeploymentID,
		Output: l.Text,
	}

	raw, err := json.Marshal(&data)
	if err != nil {
		return err
	}

	if err := s.pusher.Publish(string(raw), "log_line", channel); err != nil {
		return err
	}

	return s.logsService.LogLinesCreate(l)
}

// logWriter is an io.Writer implementation that writes log lines using a
// logsService.
type logWriter struct {
	createLogLine func(*LogLine) error

	// deployment is the deployment that the log lines will be associated
	// with.
	deploymentID string
}

// Write creates a new LogLine for each line from p.
func (w *logWriter) Write(p []byte) (int, error) {
	r := bufio.NewReader(bytes.NewReader(p))

	createLine := func(text string) error {
		return w.createLogLine(&LogLine{
			DeploymentID: w.deploymentID,
			Text:         text,
			At:           time.Now(),
		})
	}

	read := len(p)

	for {
		b, err := r.ReadBytes('\n')

		// Heroku may send a null character as a heartbeat signal. We
		// want to strip out any null characters, as inserting them into
		// postgres will cause an error.
		line := strings.Replace(string(b), "\x00", "", -1)

		if err != nil {
			if err == io.EOF {
				return read, createLine(line)
			} else {
				return read, err
			}
		}

		if err := createLine(line); err != nil {
			return read, err
		}
	}

	return read, nil
}
