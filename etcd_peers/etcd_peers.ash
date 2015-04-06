#!/bin/ash
#
# This script is meant to parse the results of an etcd discovery url and output
# nodes that appear healthy in an environment variable friendy format.
#
# Usage:
#
#     etcd_peers http://discovery.url [<env var>] [<max_results>] [<min_results>]
#
# Examples:
#
#     $ etcd_peers http://discovery.url ETCD_URLS 2
#     $ ETCD_URLS=http://ur1.com:7001,http://ur2.com:7001

syntax() {
    echo "etcd_peers http://discovery.url [<env var>] [<max_results>] [<min_results>]"
}

DISC_URL=$1
[ -z "$DISC_URL" ] && syntax && exit 1

VAR_NAME=${2:-FLEET_ETCD_SERVERS}
MAXURLS=${3:-""}
MINURLS=${4:-1}

while [ 1 ]
do
    GOODURLS=""
    URLS=$(curl $DISC_URL | jq -r '.node.nodes | map(.value) | .[]')
    # if no urls found, sleep and retry
    if [ -z "$URLS" ]
    then
        sleep 5
        continue
    fi

    # gather 'good' urls
    for url in $URLS
    do
        QUERYURL=$(echo $url | tr "7001" "4001")
        if curl -s --connect-timeout 1 $QUERYURL/v2/machines > /dev/null 2>&1
        then
            GOODURLS="$GOODURLS $QUERYURL"
        fi
        URLCOUNT=$(echo $GOODURLS | wc | awk '{print $2}')
        if [ ! -z "$URLCOUNT" -a ! -z "$MAXURLS" ]
        then
            [ "$URLCOUNT" -ge "$MAXURLS" ] && break
        fi
    done

    # If we don't have at least one, or at least $MINURLS, continue
    if [ $URLCOUNT -lt 1 -o $URLCOUNT -lt $MINURLS ]
    then
        sleep 5
        continue
    fi

    OUT=$(echo $GOODURLS | tr " " ",")
    break
done

echo "${VAR_NAME}=\"$OUT\""
