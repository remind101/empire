package etcd_peers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/coreos/go-etcd/etcd"
)

// Generated from here: http://mervine.net/json2struct
// Matches data structure from here:
//    https://discovery.etcd.io/026dd048e41a3dbb484bd02bd0b3055a
type Nodes []struct {
	CreatedIndex  float64 `json:"createdIndex"`
	Expiration    string  `json:"expiration"`
	Key           string  `json:"key"`
	ModifiedIndex float64 `json:"modifiedIndex"`
	Ttl           int32   `json:"ttl"`
	Value         string  `json:"value"`
}

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

func NodesToPeerUrls(nodes *Nodes) (*[]string, error) {
	peers := make([]string, len(*nodes))
	for i, n := range *nodes {
		parsed, err := url.Parse(n.Value)
		if err != nil {
			return nil, err
		}
		hostParts := strings.Split(parsed.Host, ":")
		peers[i] = fmt.Sprintf("http://%s:4001/", hostParts[0])
	}
	return &peers, nil

}

func FindLivePeers(urls *[]string) ([]string, error) {
	c := etcd.NewClient(*urls)
	if c.SyncCluster() {
		return c.GetCluster(), nil
	} else {
		return nil, errors.New("No live etcd cluster peers found.")
	}
}

func DiscoverEtcdNodes(discoveryURL string) (*Nodes, error) {
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

	return &d.Node.Nodes, nil
}

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
