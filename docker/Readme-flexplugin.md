# Short Description
Docker image for the FlexVolume driver portion of the IBM FlexVolume Driver for OpenStack on Power

# Full Description

## How to install
*__Note:__* This component is not intended for separate use. It is used as part of Helm charts, such as the IBM PowerVC FlexVolume Driver (ibm-powervc-k8s-volume-driver) Helm chart, for the integration of OpenStack on Power Systems into the IBM Cloud Private product.

For more information about installing the IBM PowerVC FlexVolume Driver Helm chart, see [ibm-powervc-k8s-volume-driver installation](https://www.ibm.com/support/knowledgecenter/en/SSVSPA_1.4.0/com.ibm.powervc.cloud.help.doc/powervc_icp_storage.html).

## License
View [license information](https://www.apache.org/licenses/LICENSE-2.0) for the software contained in this image.

This image uses busybox as the base image, for details view the [busybox license information](https://busybox.net/license.html).

This image contains a golang binary that includes the following packages and their dependencies.  These dependencies are licensed under the [Apache 2.0](https://www.apache.org/licenses/LICENSE-2.0) and [BSD 3-Clause](https://github.com/op/go-logging/blob/master/LICENSE) licenses.

- github.com/golang/glog
- github.com/gophercloud/gophercloud
- github.com/op/go-logging
- k8s.io/apimachinery

## More Information
For more information on IBM PowerVC, visit: [IBM PowerVC](https://www.ibm.com/systems/power/software/virtualization-management/).

For more information on IBM Cloud Private, visit: [IBM Cloud Private](https://www.ibm.com/cloud/private).
