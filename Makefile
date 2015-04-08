.PHONY: bootstrap build cmd user_data vagrant plugins

cmd:
	$(MAKE) -C empire cmd

bootstrap:
	$(MAKE) -C empire bootstrap

build:
	$(MAKE) -C empire build
	$(MAKE) -C etcd_peers build
	$(MAKE) -C logger build
	$(MAKE) -C router build

user_data:
	$(eval URL := $(shell curl -s -w '\n' https://discovery.etcd.io/new))
	sed -e "s,# discovery:,discovery:," -e "s,discovery: https://discovery.etcd.io/.*,discovery: $(URL)," cluster/user-data.template > cluster/user-data

vagrant: user_data
	vagrant destroy
	vagrant up

plugins:
	$(MAKE) -C hk-plugins all

install: plugins
	mkdir -p /usr/local/lib/hk/plugin
	cp hk-plugins/build/* /usr/local/lib/hk/plugin
	chmod +x /usr/local/lib/hk/plugin/*
	cat emp > /usr/local/bin/emp
	chmod +x /usr/local/bin/emp
