ARTIFACTDIR=../../artifacts
ARTIFACT=$(ARTIFACTDIR)/empire_base.tar.gz
PACKERLOG=$(ARTIFACTDIR)/base_packer.csv
AMIFILE=$(ARTIFACTDIR)/ami.txt

.PHONY: image check-aws clean

image: $(ARTIFACT) $(AMIFILE) push

$(ARTIFACT): packer.json scripts/base.sh
	if [ ! -d $(ARTIFACTDIR) ]; then mkdir -p $(ARTIFACTDIR); fi
	packer build -machine-readable packer.json 2>&1 | tee $(PACKERLOG)

$(AMIFILE): $(PACKERLOG)
	awk 'BEGIN { FS = "," }; /base-ubuntu-east,artifact,0,id/ {print $6}' $(PACKERLOG) | cut -f 2 -d : > $(AMIFILE)

check-aws:
ifndef AWS_ACCESS_KEY_ID
	$(error Must have AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY defined in environment)
endif

push: check-aws
	aws s3 cp $(ARTIFACT) s3://empire-image-artifacts/empire_base.tar.gz
	aws s3 cp $(AMIFILE) s3://empire-image-artifacts/ami.txt

clean:
	rm -rf build $(ARTIFACT) $(AMIFILE)
