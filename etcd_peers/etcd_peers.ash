#!/bin/ash
#
# This script is meant to parse the results of an etcd discovery url and output
# nodes that appear healthy in an environment variable friendy format.
#
# Usage:
#
#     etcd_peers http://discovery.url <env var> <num results>
#
# Examples:
#
#     $ etcd_peers http://discovery.url ETCD_URLS 2
#     $ ETCD_URLS=http://ur1.com:7001,http://ur2.com:7001

DISC_URL=$1
VAR_NAME=${2:-FLEET_ETCD_SERVERS}
NUM_URLS=${3:-100}

curl $DISC_URL | jq -r '.node.nodes | map(.value) | .[]' | while read url; do curl -s --connect-timeout 1 $url/v2/machines || echo $VAR_NAME=$url; done | head -$NUM_URLS | sed 'N;s/\n/,/'
