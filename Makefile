.PHONY: bootstrap build cmd test install

cmd:
	$(MAKE) -C empire cmd
	$(MAKE) -C cli cmd

test: cmd
	$(MAKE) -C empire test

bootstrap:
	$(MAKE) -C empire bootstrap

build:
	$(MAKE) -C empire build
	$(MAKE) -C logger build
	$(MAKE) -C relay build

install: cmd
	cat cli/emp > /usr/local/bin/emp
	chmod +x /usr/local/bin/emp
	cp cli/build/empire-plugins /usr/local/bin/empire-plugins
	chmod +x /usr/local/bin/empire-plugins
	mkdir -p /usr/local/lib/hk/plugin
	cp cli/hk-plugins/* /usr/local/lib/hk/plugin
