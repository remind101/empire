build: empire_base.tar.gz base_box/empire-base.ovf build/empire-controller.box push

empire_base.tar.gz:
	curl --location -o ./empire_base.tar.gz http://empire-image-artifacts.s3-website-us-east-1.amazonaws.com/empire_base.tar.gz

base_box/empire-base.ovf: empire_base.tar.gz
	mkdir -p base_box
	tar -C base_box -xzf empire_base.tar.gz

build/empire-controller.box: packer.json
	packer build packer.json

check-aws:
ifndef AWS_ACCESS_KEY_ID
        $(error Must have AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY defined in environment)
endif

push: check-aws
	aws s3 cp build/empire-controller.box s3://empire-image-artifacts/empire-controller.box

clean:
	rm -rf empire_base.tar.gz build base_box
