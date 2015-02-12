build: empire_base.tar.gz push

empire_base.tar.gz: packer.json
	packer build packer.json

check-aws:
ifndef AWS_ACCESS_KEY_ID
	$(error Must have AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY defined in environment)
endif

push: check-aws
	aws s3 cp empire_base.tar.gz s3://empire-image-artifacts/empire_base.tar.gz

clean:
	rm empire_base.tar.gz
