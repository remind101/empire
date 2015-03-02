// Given a discoveryURL, this is meant to continually query that till it gets
// a valid set of etcd peers.  Once it has a set (at least one) it will then
// output, either to a file or stdout, with an environment variable.
//
// Meant to be used in a systemd unit that will block the launching of fleet
// on worker/minion servers until it can join the cluster.

package main

import (
	"encoding/json"
	"flag"
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
type DiscoveryData struct {
	Action string `json:"action"`
	Node   struct {
		CreatedIndex  float64 `json:"createdIndex"`
		Dir           bool    `json:"dir"`
		Key           string  `json:"key"`
		ModifiedIndex float64 `json:"modifiedIndex"`
		Nodes         []struct {
			CreatedIndex  float64 `json:"createdIndex"`
			Expiration    string  `json:"expiration"`
			Key           string  `json:"key"`
			ModifiedIndex float64 `json:"modifiedIndex"`
			Ttl           float64 `json:"ttl"`
			Value         string  `json:"value"`
		} `json:"nodes"`
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

func FindPeers(discoveryURL string) (*[]string, error) {
	peers := make([]string, 0, 5)
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

	var hostParts []string
	for _, nodeData := range d.Node.Nodes {
		peerUrl, err := url.Parse(nodeData.Value)
		if err != nil {
			continue
		}
		hostParts = strings.Split(peerUrl.Host, ":")
		host := fmt.Sprintf("%s:%d", hostParts[0], 4001)
		peers = append(peers, host)
	}
	return &peers, nil
}

func main() {
	flag.Usage = func() {
		fmt.Printf("syntax: %s [OPTIONS] discoveryURL\n\n", os.Args[0])
		flag.PrintDefaults()
	}
	var envVar = flag.String("envVar", "FLEET_ETCD_SERVERS", "The environment variable to write.")
	var outputFile = flag.String("output", "-", "The file to dump the variable to. Setting to - dumps to stdout.")
	flag.Parse()

	discoveryURL := flag.Arg(0)
	if discoveryURL == "" {
		fmt.Printf("Error: Missing discoveryURL arg.\n\n")
		flag.Usage()
		os.Exit(1)
	}

	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	sleepTime := time.Duration(5) * time.Second

	for {
		peers, err := FindPeers(discoveryURL)
		if err != nil {
			time.Sleep(sleepTime)
			continue
		}
		if len(*peers) < 1 {
			log.Printf("Got no peers from %s. Retrying.", discoveryURL)
			time.Sleep(sleepTime)
			continue
		}

		var fd *os.File
		if *outputFile == "-" {
			fd = os.Stdout
		} else {
			fd, err = os.Create(*outputFile)
			if err != nil {
				LogErr(err, "")
				time.Sleep(sleepTime)
				continue
			}
			defer fd.Close()
		}

		// XXX: Not sure if it's worth allowing someone to use a template here
		//      but for now this should be fine.
		_, err = fd.WriteString(fmt.Sprintf("%s=\"%s\"\n", *envVar, strings.Join(*peers, ",")))
		if err != nil {
			msg := fmt.Sprintf("Unable to write to %s", *outputFile)
			LogErr(err, msg)
			time.Sleep(sleepTime)
			continue
		}
		break
	}
}
