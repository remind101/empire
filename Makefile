.PHONY: bootstrap build cmd

cmd:
	$(MAKE) -C empire cmd

bootstrap:
	$(MAKE) -C empire bootstrap

build:
	$(MAKE) -C empire build
	$(MAKE) -C etcd_peers build
