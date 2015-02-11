#!/usr/bin/env bash

echo "BUILDER TYPE: $PACKER_BUILDER_TYPE"
if [[ "$PACKER_BUILDER_TYPE" != "virtualbox"* ]]
then
    echo "# Refusing to install vagrant user in non-virtualbox build."
    exit 
fi

#Install vagrant ssh key
mkdir /home/vagrant/.ssh
wget --no-check-certificate -O authorized_keys 'https://github.com/mitchellh/vagrant/raw/master/keys/vagrant.pub'
mv authorized_keys /home/vagrant/.ssh
chown -R vagrant /home/vagrant/.ssh
chmod -R go-rwsx /home/vagrant/.ssh

#Add vagrant user to passwordless sudo
cp /etc/sudoers{,.orig}
sed -i -e 's/%sudo\s\+ALL=(ALL\(:ALL\)\?)\s\+ALL/%sudo ALL=NOPASSWD:ALL/g' /etc/sudoers
