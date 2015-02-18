#!/bin/sh

echo "RUNNING SETUP"

FILEBASE=/tmp/files

# Copy upstart scripts
sudo cp $FILEBASE/etc/init/*.conf /etc/init/

# Copy config files
sudo cp $FILEBASE/etc/nginx/nginx.conf.ctmpl /etc/nginx/nginx.conf.ctmpl
sudo cp $FILEBASE/etc/consul-template.cfg /etc/consul-template.cfg

# Install required packages
if ! which nginx
then
  echo "# Installing nginx"
  sudo apt-get install -y nginx
fi

CONSUL_TEMPLATE_VERSION=0.6.5
CONSUL_TEMPLATE_TAR=consul-template_${CONSUL_TEMPLATE_VERSION}_linux_amd64.tar.gz
if ! which consul-template
then
  echo "# Installing Consul Template"
  wget https://github.com/hashicorp/consul-template/releases/download/v${CONSUL_TEMPLATE_VERSION}/${CONSUL_TEMPLATE_TAR}
  sudo tar --strip-components=1 -xvf ${CONSUL_TEMPLATE_TAR} -C /usr/local/bin
  rm ${CONSUL_TEMPLATE_TAR}
fi