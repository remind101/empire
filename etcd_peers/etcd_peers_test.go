package etcd_peers

import (
	"os"
	"testing"

	"code.google.com/p/go-uuid/uuid"
)

func randomFileName(base string, length int) string {
	i := uuid.NewRandom()
	return base + string(i)[0:length]
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
