.PHONY: bootstrap build cmd user_data vagrant

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

install:
	# TODO: do this only for OS X
	brew install bash
	brew install gnu-sed --with-default-names

	mkdir -p /usr/local/lib/hk/plugin
	cp hk-plugins/* /usr/local/lib/hk/plugin
	cp .emprc $(HOME)/.emprc
	cat cli/apis.sh cli/main.sh > /usr/local/bin/emp
	chmod +x /usr/local/bin/emp
