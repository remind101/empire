.PHONY: bootstrap build cmd vagrant

cmd:
	$(MAKE) -C empire cmd

bootstrap:
	$(MAKE) -C empire bootstrap

build:
	$(MAKE) -C empire build
	$(MAKE) -C etcd_peers build

vagrant:
	vagrant destroy
	sed -e "s,# discovery:,discovery:," -e "s,discovery: https://discovery.etcd.io/.*,discovery: $$(curl -s -w '\n' https://discovery.etcd.io/new)," cluster/user-data.template > cluster/user-data
	vagrant up
