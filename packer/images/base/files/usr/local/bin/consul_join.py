"""
    Connects consul agent to existing members of the consul cluster
    within the current stack.

    Example agent start:
                consul agent -server -bootstrap-expect 3 -data-dir
                    /tmp/consul -node="agent-one" -bind="172.20.20.10"

"""

__author__ = 'Shiranka Miskin'

import logging
import time
import argparse

logger = logging.getLogger(__name__)

from urlparse import urljoin

import requests
from requests.exceptions import ConnectionError
from boto.ec2 import connect_to_region

DEBUG_FORMAT = ('[%(asctime)s] %(levelname)s %(name)s:%(lineno)d'
                '(%(funcName)s) - %(message)s')
INFO_FORMAT = ('[%(asctime)s] %(message)s')

ISO_8601 = '%Y-%m-%dT%H:%M:%S'


def parse_args():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('-v', '--verbose', action='count',
                        help="Can be specified multiple times for higher "
                             "levels of verbosity.")
    parser.add_argument("--static", action='append',
                        help="Force a static peer list, can be specified "
                             "multiple times.")

    return parser.parse_args()


def setup_logging(verbosity=False):
    log_level = logging.INFO
    log_format = INFO_FORMAT
    if verbosity > 0:
        log_level = logging.DEBUG
        log_format = DEBUG_FORMAT
    if verbosity < 2:
        logging.getLogger('boto').setLevel(logging.CRITICAL)
        logging.getLogger('requests').setLevel(logging.CRITICAL)

    return logging.basicConfig(format=log_format, datefmt=ISO_8601,
                               level=log_level)


def cluster_request(api_call):
    url = urljoin('http://localhost:8500', api_call)
    logger.debug("Requesting url: %s", url)
    return requests.get(url)


def check_ec2_instance():
    try:
        requests.get('http://instance-data.ec2.internal/latest/meta-data',
                     timeout=5)
        return True
    except ConnectionError:
        return False


def get_region():
    url = ('http://instance-data.ec2.internal/latest/meta-data/placement/'
           'availability-zone')
    r = requests.get(url)
    if not r:
        r.raise_for_status()
    return r.text[:-1]


def get_ec2_peers(my_ip):
    region = get_region()
    logger.debug("Getting peers in the %s region.", region)
    ec2 = connect_to_region(region)
    peers = []
    while not peers:
        res = ec2.get_all_instances(filters={'tag:Name': 'consul',
                                             'instance-state-name': 'running'})
        for reservation in res:
            for instance in reservation.instances:
                if my_ip != instance.private_ip_address:
                    peers.append(instance.private_ip_address)
    return peers


def get_vagrant_peers(my_ip):
    return ['192.168.55.11']


def get_peers(my_ip):
    if check_ec2_instance():
        logger.info("Running on EC2 instance, querying API for peers.")
        return get_ec2_peers(my_ip)
    logger.info("Not running in EC2, assuming vagrant and using static IP.")
    return get_vagrant_peers(my_ip)


def join_cluster(peers):
    joined_cluster = False
    while not joined_cluster:
        for peer in peers:
            if cluster_request('/v1/agent/join/' + peer):
                logger.info("Successfully joined peer %s.", peer)
                joined_cluster = True
                break
        if not joined_cluster:
            logger.info("Couldn't join any of the given peers, retrying.")
            time.sleep(2)


def main(args):
    consul_self = False
    while not consul_self:
        try:
            consul_self = cluster_request('/v1/agent/self')
        except:
            logger.exception("Unhandled Exception: Call to /v1/agent/self "
                             "failed: ")
        time.sleep(5)

    my_ip = consul_self.json()['Member']['Addr']

    peers = []

    while True:
        peers = args.static or get_peers(my_ip)
        if not peers:
            time.sleep(5)
            continue
        logger.info("Got peers info: %s", ", ".join(peers))
        break

    join_cluster(peers)


if __name__ == '__main__':
    args = parse_args()
    setup_logging(args.verbose)
    main(args)
