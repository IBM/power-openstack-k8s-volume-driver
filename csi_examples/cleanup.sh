oc delete daemonset ibm-powervc-csi-plugin
oc delete statefulset ibm-powervc-csi-attacher-plugin
oc delete statefulset ibm-powervc-csi-provisioner-plugin
oc delete statefulset ibm-powervc-csi-resizer-plugin

oc delete service ibm-powervc-csi-attacher-plugin
oc delete service ibm-powervc-csi-provisioner-plugin

oc delete clusterrolebinding ibm-powervc-csi-node-role
oc delete clusterrolebinding ibm-powervc-csi-provisioner-role
oc delete clusterrolebinding ibm-powervc-csi-attacher-role
oc delete clusterrolebinding ibm-powervc-csi-resizer-role

oc delete clusterrole ibm-powervc-csi-node
oc delete clusterrole ibm-powervc-csi-provisioner
oc delete clusterrole ibm-powervc-csi-attacher
oc delete clusterrole ibm-powervc-csi-resizer

oc delete serviceaccount ibm-powervc-csi-node
oc delete serviceaccount ibm-powervc-csi-provisioner
oc delete serviceaccount ibm-powervc-csi-attacher
oc delete serviceaccount ibm-powervc-csi-resizer

oc delete storageclass ibm-powervc-csi-volume-default
oc delete configmap ibm-powervc-config
oc delete csidriver ibm-powervc-csi

oc delete securitycontextconstraints csiaccess

oc delete csinode infnod-0.ocp-ppc64le-test-ce90a2.redhat.com
oc delete csinode master-0.ocp-ppc64le-test-ce90a2.redhat.com
oc delete csinode master-1.ocp-ppc64le-test-ce90a2.redhat.com
