#!/bin/sh
echo "RUNNING SETUP"

FILEBASE=/tmp/files

# Copy upstart scripts
sudo cp $FILEBASE/etc/init/*.conf /etc/init/
