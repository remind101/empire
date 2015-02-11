package consulutil

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/hashicorp/consul/api"

	"testing"
	"time"
)

var consulConfig = `{
	"ports": {
		"dns": 19000,
		"http": 18800,
		"rpc": 18600,
		"serf_lan": 18200,
		"serf_wan": 18400,
		"server": 18000
	},
	"bind_addr": "127.0.0.1",
	"data_dir": "%s",
	"bootstrap": true,
	"log_level": "debug",
	"server": true
}`

type testServer struct {
	pid        int
	dataDir    string
	configFile string
}

type testPortConfig struct {
	DNS     int `json:"dns,omitempty"`
	HTTP    int `json:"http,omitempty"`
	RPC     int `json:"rpc,omitempty"`
	SerfLan int `json:"serf_lan,omitempty"`
	SerfWan int `json:"serf_wan,omitempty"`
	Server  int `json:"server,omitempty"`
}

type testAddressConfig struct {
	HTTP string `json:"http,omitempty"`
}

type testServerConfig struct {
	Bootstrap bool               `json:"bootstrap,omitempty"`
	Server    bool               `json:"server,omitempty"`
	DataDir   string             `json:"data_dir,omitempty"`
	LogLevel  string             `json:"log_level,omitempty"`
	Addresses *testAddressConfig `json:"addresses,omitempty"`
	Ports     testPortConfig     `json:"ports,omitempty"`
}

// Callback functions for modifying config
type configCallback func(c *api.Config)
type serverConfigCallback func(c *testServerConfig)

func defaultConfig() *testServerConfig {
	return &testServerConfig{
		Bootstrap: true,
		Server:    true,
		LogLevel:  "debug",
		Ports: testPortConfig{
			DNS:     19000,
			HTTP:    18800,
			RPC:     18600,
			SerfLan: 18200,
			SerfWan: 18400,
			Server:  18000,
		},
	}
}

func (s *testServer) Stop() {
	defer os.RemoveAll(s.dataDir)
	defer os.RemoveAll(s.configFile)

	cmd := exec.Command("kill", "-9", fmt.Sprintf("%d", s.pid))
	if err := cmd.Run(); err != nil {
		panic(err)
	}
}

func NewTestServer(t *testing.T) *testServer {
	return NewTestServerWithConfig(t, func(c *testServerConfig) {})
}

func NewTestServerWithConfig(t *testing.T, cb serverConfigCallback) *testServer {
	if path, err := exec.LookPath("consul"); err != nil || path == "" {
		t.Log("consul not found on $PATH, skipping")
		t.SkipNow()
	}

	pidFile, err := ioutil.TempFile("", "consul")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	pidFile.Close()
	os.Remove(pidFile.Name())

	dataDir, err := ioutil.TempDir("", "consul")
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	configFile, err := ioutil.TempFile("", "consul")
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	consulConfig := defaultConfig()
	consulConfig.DataDir = dataDir

	cb(consulConfig)

	configContent, err := json.Marshal(consulConfig)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if _, err := configFile.Write(configContent); err != nil {
		t.Fatalf("err: %s", err)
	}
	configFile.Close()

	// Start the server
	cmd := exec.Command("consul", "agent", "-config-file", configFile.Name())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("err: %s", err)
	}

	return &testServer{
		pid:        cmd.Process.Pid,
		dataDir:    dataDir,
		configFile: configFile.Name(),
	}
}

func MakeClient(t *testing.T) (*api.Client, *testServer) {
	return MakeClientWithConfig(t, func(c *api.Config) {
		c.Address = "127.0.0.1:18800"
	}, func(c *testServerConfig) {})
}

func MakeClientWithConfig(t *testing.T, cb1 configCallback, cb2 serverConfigCallback) (*api.Client, *testServer) {
	// Make client config
	conf := api.DefaultConfig()
	cb1(conf)

	// Create client
	client, err := api.NewClient(conf)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Create server
	server := NewTestServerWithConfig(t, cb2)

	// Allow the server some time to start, and verify we have a leader.
	waitForResult(func() (bool, error) {
		_, qm, err := client.Catalog().Nodes(nil)
		if err != nil {
			return false, err
		}

		// Ensure we have a leader and a node registeration
		if !qm.KnownLeader {
			return false, fmt.Errorf("Consul leader status: false")
		}
		if qm.LastIndex == 0 {
			return false, fmt.Errorf("Consul index is 0")
		}

		return true, nil
	}, func(err error) {
		t.Fatalf("err: %s", err)
	})

	return client, server
}

type testFn func() (bool, error)
type errorFn func(error)

func waitForResult(test testFn, error errorFn) {
	retries := 1000

	for retries > 0 {
		time.Sleep(10 * time.Millisecond)
		retries--

		success, err := test()
		if success {
			return
		}

		if retries == 0 {
			error(err)
		}
	}
}
