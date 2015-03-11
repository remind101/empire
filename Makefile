.PHONY: bootstrap build cmd user_data vagrant

cmd:
	$(MAKE) -C empire cmd

bootstrap:
	$(MAKE) -C empire bootstrap

build:
	docker build --no-cache -t empire .

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
