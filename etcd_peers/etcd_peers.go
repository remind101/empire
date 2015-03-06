package etcd_peers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// Generated from here: http://mervine.net/json2struct
// Matches data structure from here:
//    https://discovery.etcd.io/026dd048e41a3dbb484bd02bd0b3055a
type Node struct {
	CreatedIndex  float64 `json:"createdIndex"`
	Expiration    string  `json:"expiration"`
	Key           string  `json:"key"`
	ModifiedIndex float64 `json:"modifiedIndex"`
	Ttl           int32   `json:"ttl"`
	Value         string  `json:"value"`
}

type Nodes []Node

type DiscoveryData struct {
	Action string `json:"action"`
	Node   struct {
		CreatedIndex  float64 `json:"createdIndex"`
		Dir           bool    `json:"dir"`
		Key           string  `json:"key"`
		ModifiedIndex float64 `json:"modifiedIndex"`
		Nodes         Nodes   `json:"nodes"`
	} `json:"node"`
}

// Outputs errors in a standard way.
func LogErr(err error, msg string) {
	if msg == "" {
		log.Printf("Error: %s\n", err)
	} else {
		log.Printf("Error: %s\n", msg)
		log.Printf("       %s\n", err)
	}
}

// Takes a node list from the discoveryURL, which includes the server peer
// address and turns them into client URLs.
func NodesToClientUrls(nodes Nodes) ([]string, error) {
	peers := make([]string, len(nodes))
	for i, n := range nodes {
		parsed, err := url.Parse(n.Value)
		if err != nil {
			return nil, err
		}
		hostParts := strings.Split(parsed.Host, ":")
		peers[i] = fmt.Sprintf("http://%s:4001/", hostParts[0])
	}
	return peers, nil

}

// Given a list of client URLs, connect to each to and return a slice of which
// is actually serving.
func FindLivePeers(urls []string, count int, schema string) ([]string, error) {
	peers := make([]string, 0, len(urls))
	c := http.Client{Timeout: time.Second}
	for _, u := range urls {
		parsed, err := url.Parse(u)
		if err != nil {
			continue
		}
		// Easiest path to ping and see if the etcd daemon is responsive
		parsed.Path = "/v2/machines"
		_, err = c.Get(parsed.String())
		if err != nil {
			continue
		}

		if schema != "" {
			parsed.Scheme = schema
		}
		// Need to return back to the root for the actual output
		parsed.Path = "/"

		peers = append(peers, parsed.String())
		if count > 0 && len(peers) >= count {
			break
		}
	}
	return peers, nil
}

// Connects to an etcd discoveryURL and grabs the list of nodes registered
func DiscoverEtcdNodes(discoveryURL string) (Nodes, error) {
	client := http.Client{}
	resp, err := client.Get(discoveryURL)
	if err != nil {
		msg := fmt.Sprintf("Problem with GET of %s:", discoveryURL)
		LogErr(err, msg)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		msg := fmt.Sprintf("Unable to read body from %s:", discoveryURL)
		LogErr(err, msg)
		return nil, err
	}

	var d DiscoveryData
	err = json.Unmarshal(body, &d)
	if err != nil {
		LogErr(err, "JSON unmarshaling error")
		return nil, err
	}

	return d.Node.Nodes, nil
}

// Creates/opens a file to write to it, or uses stdin if '-' is given.
func GetOutput(oFile string) (*os.File, error) {
	var fd *os.File
	if oFile == "-" {
		fd = os.Stdout
		return fd, nil
	} else {
		fd, err := os.Create(oFile)
		if err != nil {
			return nil, err
		}
		return fd, nil
	}
}
