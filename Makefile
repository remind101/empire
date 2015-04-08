.PHONY: bootstrap build cmd user_data vagrant plugins cli install

cmd:
	$(MAKE) -C empire cmd

bootstrap:
	$(MAKE) -C empire bootstrap

build:
	$(MAKE) -C empire build
	$(MAKE) -C logger build
	$(MAKE) -C relay build

plugins:
	$(MAKE) -C cli cmd
	cp cli/build/empire-plugins /usr/local/bin/empire-plugins
	chmod +x /usr/local/bin/empire-plugins
	mkdir -p /usr/local/lib/hk/plugin
	cp hk-plugins/* /usr/local/lib/hk/plugin

cli:
	cat emp > /usr/local/bin/emp
	chmod +x /usr/local/bin/emp

install: cli plugins