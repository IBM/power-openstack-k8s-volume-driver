#!/bin/sh
# Copyright IBM Corp. 2018.
#
#  Licensed under the Apache License, Version 2.0 (the "License");
#  you may not use this file except in compliance with the License.
#  You may obtain a copy of the License at
#      http://www.apache.org/licenses/LICENSE-2.0
#
#  Unless required by applicable law or agreed to in writing, software
#  distributed under the License is distributed on an "AS IS" BASIS,
#  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#  See the License for the specific language governing permissions and
#  limitations under the License.

set -o errexit
set -o pipefail

VENDOR=ibm
PREFIX=${1:-power-openstack-k8s}
DRIVER=$PREFIX-volume-flex

# Kubernetes expects the flex volume driver to be in a vendor~driver directory
driver_dir=$VENDOR${VENDOR:+"~"}${DRIVER}
if [ ! -d "/flex-mount-dir/$driver_dir" ]; then
  /bin/mkdir "/flex-mount-dir/$driver_dir"
fi

# We need to write out the environment variables to a config file for the driver to access
/bin/cat > "/flex-mount-dir/$driver_dir/$DRIVER.conf" <<EOF
OS_AUTH_URL=$OS_AUTH_URL
OS_USERNAME=$OS_USERNAME
OS_PASSWORD_ENC=`/bin/echo $OS_PASSWORD | /bin/base64`
OS_DOMAIN_NAME=$OS_DOMAIN_NAME
OS_PROJECT_NAME=$OS_PROJECT_NAME
EOF

# Make sure we restrict access to the configuration file that we have written out
/bin/chmod 600 "/flex-mount-dir/$driver_dir/$DRIVER.conf"

# If the user gave us a certificate to use, then lets just put it in the driver directory
if [[ ! -z "$OS_CACERT" ]]; then
   /bin/cp -f $OS_CACERT "/flex-mount-dir/$driver_dir/$DRIVER.crt"
   /bin/echo "OS_CACERT=/usr/libexec/kubernetes/kubelet-plugins/volume/exec/$driver_dir/$DRIVER.crt"  >> "/flex-mount-dir/$driver_dir/$DRIVER.conf"
fi

# To make the operation more atomic we will first copy it to a temporary location and then move it
/bin/cp -f "/power-openstack-k8s-volume-flex" "/flex-mount-dir/$driver_dir/.$DRIVER"
/bin/mv -f "/flex-mount-dir/$driver_dir/.$DRIVER" "/flex-mount-dir/$driver_dir/$DRIVER"

# We just need the container to keep running for the daemon set even though it is done now
while : ; do
  sleep 30
done
