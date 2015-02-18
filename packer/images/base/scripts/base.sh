#!/bin/bash

FILEBASE=/tmp/files

sudo apt-get update
sudo apt-get -y install wget unzip curl vim-nox

if ! which pip > /dev/null 2>&1
then
    echo "# Installing pip"
    curl https://bootstrap.pypa.io/get-pip.py | sudo python
fi

sudo -H pip install httpie
sudo -H pip install virtualenv
sudo -H pip install boto
sudo -H pip install requests

if ! which docker
then
    echo "# Installing docker"
    sudo apt-get -y install docker.io
    sudo service docker.io stop
    wget https://get.docker.com/builds/Linux/x86_64/docker-latest -O /tmp/docker
    sudo cp /tmp/docker /usr/bin
    rm /tmp/docker
    sudo usermod -a -G docker vagrant
    sudo usermod -a -G docker ubuntu
fi

CONSULZIP=0.4.1_linux_amd64.zip
if ! which consul
then
    echo "# Installing consul"
    wget https://dl.bintray.com/mitchellh/consul/${CONSULZIP} && sudo unzip -d /usr/local/bin ${CONSULZIP}
    rm ${CONSULZIP}
fi

cp $FILEBASE/usr/local/bin/consul_join.py /usr/local/bin/consul_join
chmod +x /usr/local/bin/consul_join

cp $FILEBASE/etc/init/*.conf /etc/init

# Remove old FILEBASE
rm -rf $FILEBASE
