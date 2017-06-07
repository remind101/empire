package honeybadger

import (
	"encoding/json"
	"errors"
	"testing"
)

func newTestError() Error {
	var frames []*Frame
	frames = append(frames, &Frame{
		File:   "/path/to/root/badgers.go",
		Number: "1",
		Method: "badgers",
	})
	frames = append(frames, &Frame{
		File:   "/foo/bar/baz.go",
		Number: "2",
		Method: "baz",
	})
	return Error{
		err:     errors.New("Cobras!"),
		Message: "Cobras!",
		Class:   "honeybadger",
		Stack:   frames,
	}
}

func TestNewNotice(t *testing.T) {
	err := newTestError()
	var notice *Notice

	notice = newNotice(&Configuration{Root: "/path/to/root"}, err)

	if notice.ErrorMessage != "Cobras!" {
		t.Errorf("Unexpected value for notice.ErrorMessage. expected=%#v result=%#v", "Cobras!", notice.ErrorMessage)
	}

	if notice.Error.err != err.err {
		t.Errorf("Unexpected value for notice.Error. expected=%#v result=%#v", err.err, notice.Error.err)
	}

	if notice.Backtrace[0].File != "[PROJECT_ROOT]/badgers.go" {
		t.Errorf("Expected notice to substitute project root. expected=%#v result=%#v", "[PROJECT_ROOT]/badgers.go", notice.Backtrace[0].File)
	}

	if notice.Backtrace[1].File != "/foo/bar/baz.go" {
		t.Errorf("Expected notice not to trash non-project file. expected=%#v result=%#v", "/foo/bar/baz.go", notice.Backtrace[1].File)
	}

	notice = newNotice(&Configuration{Root: ""}, err)
	if notice.Backtrace[0].File != "/path/to/root/badgers.go" {
		t.Errorf("Expected notice not to trash project root. expected=%#v result=%#v", "/path/to/root/badgers.go", notice.Backtrace[0].File)
	}
}

func TestToJSON(t *testing.T) {
	notice := newNotice(Config, newError(errors.New("Cobras!"), 0))
	raw := notice.toJSON()

	var payload hash
	err := json.Unmarshal(raw, &payload)
	if err != nil {
		t.Errorf("Got error while parsing notice JSON err=%#v json=%#v", err, raw)
		return
	}

	testNoticePayload(t, payload)
}
