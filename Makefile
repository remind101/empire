.PHONY: bootstrap build cmd test user_data vagrant

cmd:
	$(MAKE) -C empire cmd
	$(MAKE) -C etcd_peers cmd

bootstrap:
	$(MAKE) -C empire bootstrap

build:
	$(MAKE) -C empire build
	$(MAKE) -C etcd_peers build

test:
	$(MAKE) -C empire test
	$(MAKE) -C etcd_peers test

user_data:
	$(eval URL := $(shell curl -s -w '\n' https://discovery.etcd.io/new))
	sed -e "s,# discovery:,discovery:," -e "s,discovery: https://discovery.etcd.io/.*,discovery: $(URL)," cluster/user-data.template > cluster/user-data

vagrant: user_data
	vagrant destroy
	vagrant up

install:
	mkdir -p /usr/local/lib/hk/plugin
	cp hk-plugins/* /usr/local/lib/hk/plugin
	cat emp > /usr/local/bin/emp
	chmod +x /usr/local/bin/emp
