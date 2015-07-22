package tugboat

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestLogWriter(t *testing.T) {
	tests := []struct {
		in  io.Reader
		out []string
	}{
		{
			strings.NewReader(`-----> Fetching custom git buildpack... done
-----> Go app detected
-----> Using go1.3
-----> Running: godep go install -tags heroku ./...
-----> Discovering process types
       Procfile declares types -> web

-----> Compressing... done, 1.6MB
-----> Launching... done, v6
       https://acme-inc.herokuapp.com/ deployed to Heroku
`),
			[]string{
				"-----> Fetching custom git buildpack... done\n",
				"-----> Go app detected\n",
				"-----> Using go1.3\n",
				"-----> Running: godep go install -tags heroku ./...\n",
				"-----> Discovering process types\n",
				"       Procfile declares types -> web\n",
				"\n",
				"-----> Compressing... done, 1.6MB\n",
				"-----> Launching... done, v6\n",
				"       https://acme-inc.herokuapp.com/ deployed to Heroku\n",
				"",
			},
		},
		{
			bytes.NewReader([]byte{
				'h', 'e', 'l', 'l', 'o', '\n',
				0x00,
				'w', 'o', 'r', 'l', 'd',
			}),
			[]string{
				"hello\n",
				"world",
			},
		},
	}

	for _, tt := range tests {
		var lines []*LogLine

		w := &logWriter{
			createLogLine: func(l *LogLine) error {
				lines = append(lines, l)
				return nil
			},
			deploymentID: "1",
		}

		if _, err := io.Copy(w, tt.in); err != nil {
			t.Fatal(err)
		}

		if got, want := len(lines), len(tt.out); got != want {
			t.Fatalf("len(lines) => %d; want %d", got, want)
		}

		for i := 0; i < len(lines); i++ {
			if got, want := lines[i].Text, tt.out[i]; got != want {
				t.Errorf("Line #%d => %q; want %q", i, got, want)
			}
		}
	}
}
