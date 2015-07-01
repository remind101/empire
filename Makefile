.PHONY: bootstrap build cmd test

cmd:
	$(MAKE) -C empire cmd

test: cmd
	$(MAKE) -C empire test

bootstrap:
	$(MAKE) -C empire bootstrap

build:
	$(MAKE) -C empire build
