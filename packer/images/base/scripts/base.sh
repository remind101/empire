#!/bin/bash

sudo apt-get update
sudo apt-get -y install wget unzip curl

if ! pip --help > /dev/null
then
    echo "# Installing pip"
    curl https://bootstrap.pypa.io/get-pip.py | sudo python
fi

sudo pip install httpie
sudo pip install virtualenv

if ! which docker
then
    echo "# Installing docker"
    sudo apt-get -y install docker.io
    sudo service docker.io stop
    wget https://get.docker.com/builds/Linux/x86_64/docker-latest -O /tmp/docker
    sudo cp /tmp/docker /usr/bin
    sudo usermod -a -G docker vagrant
    sudo usermod -a -G docker ubuntu
fi

CONSULZIP=0.4.1_linux_amd64.zip
if ! which consul
then
    echo "# Installing consul"
    wget https://dl.bintray.com/mitchellh/consul/${CONSULZIP} && sudo unzip -d /usr/local/bin ${CONSULZIP}
fi
