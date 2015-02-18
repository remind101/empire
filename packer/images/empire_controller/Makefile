BASENAME = empire_base.tar.gz
BASEIMAGE = build/$(BASENAME)
IMAGENAME = empire_controller
AMIFILE = ami.txt
AMIPATH = build/$(AMIFILE)
BOXNAME = $(IMAGENAME).box
ARTIFACT = ../../artifacts/$(BOXNAME)
BTSYNCDIR = $(HOME)/Documents/remind_sync
BTSYNCFILE = $(BTSYNCDIR)/$(BOXNAME)

.PHONY: image check-aws clean push

image: build $(BASEIMAGE) $(AMIPATH) build/empire-base.ovf $(ARTIFACT) $(BTSYNCFILE) push

build:
	mkdir -p build

$(BASEIMAGE): ../../artifacts/$(BASENAME)
	if [ -f ../../artifacts/$(BASENAME) ]; then cp ../../artifacts/$(BASENAME) $(BASEIMAGE); else curl -f --location -o ./$(BASEIMAGE) http://empire-image-artifacts.s3-website-us-east-1.amazonaws.com/$(BASENAME); fi

$(AMIPATH): ../../artifacts/$(AMIFILE)
	if [ -f ../../artifacts/$(AMIFILE) ]; then cp ../../artifacts/$(AMIFILE) $(AMIPATH); else  curl -f --location -o ./$(AMIPATH) http://empire-image-artifacts.s3-website-us-east-1.amazonaws.com/$(AMIFILE); fi

build/empire-base.ovf: $(BASEIMAGE)
	mkdir -p build
	tar -C build -xzf $(BASEIMAGE)

$(ARTIFACT): packer.json
	packer build --var "baseami=`cat $(AMIPATH)`" packer.json

check-aws:
ifndef AWS_ACCESS_KEY_ID
        $(error Must have AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY defined in environment)
endif

push: check-aws
	aws s3 cp $(ARTIFACT) s3://empire-image-artifacts/$(BOXNAME)

$(BTSYNCFILE): $(ARTIFACT)
	cp $(ARTIFACT) $(BTSYNCDIR)

clean:
	rm -rf build $(ARTIFACT)
