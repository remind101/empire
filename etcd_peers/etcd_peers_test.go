package etcd_peers

import (
	"math/rand"
	"os"
	"testing"
)

var letters = []rune("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randomFileName(base string, length int) string {
	b := make([]rune, length)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return base + string(b)
}

func TestGetOutputStdout(t *testing.T) {
	fd, err := GetOutput("-")
	if err != nil {
		panic(err)
	}
	defer fd.Close()
	defer os.Remove(fd.Name())

	name := fd.Name()
	if err != nil {
		panic(err)
	}

	if name != "/dev/stdout" {
		t.Errorf("getOutput(\"-\") did not return stdout, instead returned: %s", name)
	}
}

func TestGetOutputFile(t *testing.T) {
	fname := randomFileName("/tmp/.etcd_peers_test.", 8)
	fd, err := GetOutput(fname)
	if err != nil {
		panic(err)
	}
	defer fd.Close()
	defer os.Remove(fd.Name())

	name := fd.Name()
	if err != nil {
		panic(err)
	}

	if name != fname {
		t.Errorf("getOutput(%s) did not return %s, instead returned: %s", fname, fname, name)
	}
}
