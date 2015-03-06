// Given a discoveryURL, this is meant to continually query that till it gets
// a valid set of etcd peers.  Once it has a set (at least one) it will then
// output, either to a file or stdout, with an environment variable.
//
// Meant to be used in a systemd unit that will block the launching of fleet
// on worker/minion servers until it can join the cluster.

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/remind101/empire/etcd_peers"
)

func main() {
	flag.Usage = func() {
		fmt.Printf("syntax: %s [OPTIONS] discoveryURL\n\n", os.Args[0])
		flag.PrintDefaults()
	}
	var envVar = flag.String("envVar", "FLEET_ETCD_SERVERS", "The environment variable to write.")
	var outputFile = flag.String("output", "-", "The file to dump the variable to. Setting to - dumps to stdout.")
	var onePeer = flag.Bool("1", false, "If set, only dump the peer with the longest TTL (the most recent).")
	var sleep = flag.Duration("sleep", time.Duration(5)*time.Second, "Time to sleep between attempts to discover nodes.")
	var schema = flag.String("schema", "", "Replace the schema of each peer with this string. Useful for things like registrator that use etcd instead of http to determine the schema.")
	flag.Parse()

	discoveryURL := flag.Arg(0)
	if discoveryURL == "" {
		fmt.Printf("Error: Missing discoveryURL arg.\n\n")
		flag.Usage()
		os.Exit(1)
	}

	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	for {
		nodes, err := etcd_peers.DiscoverEtcdNodes(discoveryURL)
		if err != nil {
			time.Sleep(*sleep)
			continue
		}
		if len(nodes) < 1 {
			log.Printf("Got no peers from %s. Retrying.", discoveryURL)
			time.Sleep(*sleep)
			continue
		}
		urls, err := etcd_peers.NodesToClientUrls(nodes)
		if err != nil {
			etcd_peers.LogErr(err, "Error transforming peers.")
			time.Sleep(*sleep)
			continue
		}
		// 0 means no max number of hosts, return all that are alive
		count := 0
		if *onePeer {
			count = 1
		}

		livePeers, err := etcd_peers.FindLivePeers(urls, count, *schema)

		if len(livePeers) < 1 {
			log.Println("No live peers, continuing.")
			time.Sleep(*sleep)
			continue
		}

		fd, err := etcd_peers.GetOutput(*outputFile)
		if err != nil {
			msg := fmt.Sprintf("Unable to open -output '%s'", *outputFile)
			etcd_peers.LogErr(err, msg)
			time.Sleep(*sleep)
			continue
		}

		_, err = fd.WriteString(fmt.Sprintf("%s=\"%s\"\n", *envVar, strings.Join(livePeers, ",")))
		if err != nil {
			msg := fmt.Sprintf("Unable to write to %s", *outputFile)
			etcd_peers.LogErr(err, msg)
			time.Sleep(*sleep)
			continue
		}
		break
	}
}
