#!/bin/sh

set -o errexit
set -o pipefail

## VENDOR and DRIVER are fixed in the Dockerfile.
## Can be defined from the container runtime.

# Assuming the single driver file is located at /$DRIVER inside the DaemonSet image.

driver_dir=$VENDOR${VENDOR:+"~"}${DRIVER}

echo 'Installing driver '$driver_dir'/'$DRIVER

if [ ! -d "/flexmnt/$driver_dir" ]; then
  mkdir "/flexmnt/$driver_dir"
fi

cp "/smb-flex-driver" "/flexmnt/$driver_dir/.$DRIVER"
mv -f "/flexmnt/$driver_dir/.$DRIVER" "/flexmnt/$driver_dir/$DRIVER"

chmod +x "/flexmnt/$driver_dir/$DRIVER"

echo '
   _       _ _       _                  __   _  __
  (_)_   _| (_) ___ | |__  _ __ ___    / /__(_)/ _|___
  | | | | | | |/ _ \| '_ \| '_ ` _ \  / / __| | |_/ __|
  | | |_| | | | (_) | | | | | | | | |/ / (__| |  _\__ \
 _/ |\__,_|_|_|\___/|_| |_|_| |_| |_/_/ \___|_|_| |___/
|__/
Driver has been installed.
Make sure /flexmnt from this container mounts to Kubernetes driver directory.
  k8s 1.8.x
  /usr/libexec/kubernetes/kubelet-plugins/volume/exec/
This path may be different in your system due to kubelet parameter --volume-plugin-dir.
This driver depends on the following packages to be installed on the host:
  ## ubuntu
  apt-get install -y cifs-utils
  ## centos
  yum install -y cifs-utils
This container can now be stopped and removed.
'

while : ; do
  sleep 3600
done
