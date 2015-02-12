#!/bin/sh

ARCH=$(uname)
BINDIR=$HOME/bin

if [ "$ARCH" != "Darwin" ]
then
    echo "# Only works in MacOS. Aborting."
    exit 1
fi

if [ ! -f ${BINDIR}/packer ]
then
    if [ ! -d $BINDIR ]
    then
        echo "# $BINDIR does not exist. Creating."
        mkdir -p $HOME/bin
        echo "# Be sure to add $BINDIR to your PATH."
        echo "# ex. export PATH=$PATH:$BINDIR"
    fi

    echo "# Installing packer."
    PACKER=https://dl.bintray.com/mitchellh/packer/packer_0.7.5_darwin_amd64.zip
    curl --location -o /tmp/packer.zip $PACKER
    cd $BINDIR
    unzip /tmp/packer.zip
fi

if ! which aws > /dev/null 2>&1
then
    if ! which pip > /dev/null 2>&1
    then
        echo "# Installing pip"
        curl https://bootstrap.pypa.io/get-pip.py | sudo python
    fi
    echo "# Installing aws cli"
    sudo pip install awscli
fi

if ! which vagrant > /dev/null 2>&1
then
    echo "# Installing vagrant."
    curl --location -o /tmp/vagrant.dmg https://dl.bintray.com/mitchellh/vagrant/vagrant_1.7.2.dmg
    echo "# Opening the vagrant .dmg file. Please double click 'Vagrant.pkg' "
    echo "# when it is finished, and follow the instructions to install."
    open /tmp/vagrant.dmg
    echo "# Hit ENTER when you are done installing to continue."
    sleep 15
    read CONTINUE
fi

if ! vagrant plugin list | grep -q vagrant-vbguest
then
    echo "# Installing vagrant-vbguest plugin."
    vagrant plugin install vagrant-vbguest
fi

echo "# Done"
