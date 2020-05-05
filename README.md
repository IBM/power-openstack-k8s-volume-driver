# IBM FlexVolume Driver for OpenStack on Power


This project contains the source code to build the docker images for the FlexVolume driver and volume provisioner to integrate OpenStack on Power storage into the IBM Cloud Private product.  The IBM FlexVolume driver provides the support within IBM Cloud Private to communicate with OpenStack on Power to provision persistent volumes and attach those volumes to worker nodes so that they may be mounted to containers.

There are 2 separate docker images created as part of this project, the *__ibmcom/power-openstack-k8s-volume-flex__* and *__ibmcom/power-openstack-k8s-volume-provisioner__*, where the volume-provisioner provides an external storage provisioner in Kubernetes to create and delete volumes and the volume-flex provides a Kubernetes flex volume driver that handles the attaching/detaching volumes to the worker nodes and mounting the appropriate directories.

These images are not meant to be used standalone but rather deployed in coordination through a helm chart, such as the *__ibm-powervc-k8s-volume-driver__* helm chart, into the IBM Cloud Private product.  These images will be pulled in automatically when the helm chart is installed.

# Build
  In summary, the build is driven by installing the appropriate golang packages (through glade) on the system and basic docker support, and executing the makefile to compile the go programs and build and save the docker images. 

# Install
When using ICp, the docker images will be available on docker hub so will be implicitly pulled and loaded as part of the helm chart installation, but for development and test an additional step is needed.  

  docker image load < $TMP_DIR/power-openstack-k8s-volume-driver-img-1.0.0.tar.gz

# Test
To test these docker images, first the docker images must be loaded through the mechanism described in the install step.  Once the images are loaded then the *__ibm-powervc-k8s-volume-driver__* helm chart must be installed so that the flex driver and provisioner are registered within Kubernetes.  From this point a persistent volume claim can be created, using the *__ibm-powervc-k8s-volume-default__* storage class, and then pods/containers can be deployed using this persistent volume claim to mount storage to the given containers.


# IBM PowerVC CSI Driver

**Knowledge Center Documentation:**

[Installation/Configuration Steps](http://blaze.aus.stglabs.ibm.com/kc20A-bld/SSXK2N_1.4.4/com.ibm.powervc.standard.help.doc/powervc_csi_storage_install.html)

**Templates**

The following 3 templates for PowerVC CSI Driver are available here:

- [ibm-powervc-csi-driver-template.yaml](templates/ibm-powervc-csi-driver-template.yaml)
- [scc.yaml](templates/scc.yaml)
- [secret.yaml](secret.yaml)

**Sample scripts are available here:**

[Examples](csi_examples)

