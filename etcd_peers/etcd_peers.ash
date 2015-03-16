#!/bin/ash
#
# Usage: etcd_peers http://discovery.url <env var> <num results>
DISC_URL=$1
VAR_NAME=${2:-FLEET_ETCD_SERVERS}
NUM_URLS=${3:-100}

curl $DISC_URL | jq -r '.node.nodes | map(.value) | .[]' | while read url; do curl -s --connect-timeout 1 $url/v2/machines || echo $VAR_NAME=$url; done | head -$NUM_URLS | sed 'N;s/\n/,/'
