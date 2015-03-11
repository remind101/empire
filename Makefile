.PHONY: bootstrap build cmd user_data_vulcand user_data_nginx vagrant

cmd:
	$(MAKE) -C empire cmd

bootstrap:
	$(MAKE) -C empire bootstrap

build:
	$(MAKE) -C empire build
	$(MAKE) -C etcd_peers build

user_data_nginx:
	$(eval URL := $(shell curl -s -w '\n' https://discovery.etcd.io/new))
	sed -e "s,# discovery:,discovery:," -e "s,discovery: https://discovery.etcd.io/.*,discovery: $(URL)," cluster/user-data_nginx.template > cluster/user-data

user_data_vulcand:
	$(eval URL := $(shell curl -s -w '\n' https://discovery.etcd.io/new))
	sed -e "s,# discovery:,discovery:," -e "s,discovery: https://discovery.etcd.io/.*,discovery: $(URL)," cluster/user-data_vulcand.template > cluster/user-data

vagrant_nginx: user_data_nginx
	vagrant destroy
	vagrant up

vagrant_vulcand: user_data_vulcand
	vagrant destroy
	vagrant up

install:
	mkdir -p /usr/local/lib/hk/plugin
	cp hk-plugins/* /usr/local/lib/hk/plugin
	cat emp > /usr/local/bin/emp
	chmod +x /usr/local/bin/emp
