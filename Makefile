.PHONY: bootstrap build cmd

cmd:
	$(MAKE) -C empire cmd

bootstrap:
	$(MAKE) -C empire bootstrap

build:
	$(MAKE) -C empire build
	$(MAKE) -C logger build
	$(MAKE) -C relay build

install:
	mkdir -p /usr/local/lib/hk/plugin
	cp hk-plugins/* /usr/local/lib/hk/plugin
	cat emp > /usr/local/bin/emp
	chmod +x /usr/local/bin/emp
