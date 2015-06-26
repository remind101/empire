// +build docker

package runner

import (
	"bytes"
	"strings"
	"testing"

	"golang.org/x/net/context"

	"github.com/remind101/empire/empire/pkg/dockerutil"
	"github.com/remind101/empire/empire/pkg/image"
)

func TestRunner(t *testing.T) {
	r := newTestRunner(t)
	out := new(bytes.Buffer)

	if err := r.Run(context.Background(), RunOpts{
		Image: image.Image{
			Repository: "ubuntu",
			Tag:        "14.04",
		},
		Command: "/bin/bash",
		Input:   strings.NewReader("ls\nexit\n"),
		Output:  out,
	}); err != nil {
		t.Fatal(err)
	}
}

func newTestRunner(t testing.TB) *Runner {
	c, err := dockerutil.NewClientFromEnv(nil)
	if err != nil {
		t.Fatal(err)
	}

	return NewRunner(c)
}
