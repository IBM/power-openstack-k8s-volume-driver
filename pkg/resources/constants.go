/*
  Copyright IBM Corp. 2018, 2019.

  Licensed under the Apache License, Version 2.0 (the "License");
  you may not use this file except in compliance with the License.
  You may obtain a copy of the License at
      http://www.apache.org/licenses/LICENSE-2.0

  Unless required by applicable law or agreed to in writing, software
  distributed under the License is distributed on an "AS IS" BASIS,
  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  See the License for the specific language governing permissions and
  limitations under the License.
*/
package resources

import (
	"fmt"
	"os"
)

// Constants
const (
	OpInit          = "init"
	OpGetVolumeName = "getvolumename"
	OpIsAttached    = "isattached"
	OpAttach        = "attach"
	OpWaitForAttach = "waitforattach"
	OpMountDevice   = "mountdevice"
	OpMount         = "mount"
	OpDetach        = "detach"
	OpWaitForDetach = "waitfordetach"
	OpUnmountDevice = "unmountdevice"
	OpUnmount       = "unmount"

	// Operation params
	NodeName       = "nodename"
	DevicePath     = "devicePath"
	MountPath      = "mountPath"
	MountDir       = "mountDir"
	DriverJSONArgs = "jsonArgs"

	// Kuberenets args
	K8sArgFSType   = "kubernetes.io/fsType"
	K8sArgMountRW  = "kubernetes.io/readwrite"
	K8sArgSecret   = "kubernetes.io/secret"
	K8sArgFSGroup  = "kubernetes.io/fsGroup"
	K8sArgMountDir = "kubernetes.io/mountsDir"
	K8sArgPV       = "kubernetes.io/pvOrVolumeName"
	K8sCreatedBy   = "kubernetes.io/createdby"

	// Openstack args
	OsArgsVolID         = "volumeID"
	OsArgsVolWWN        = "wwn"
	OsArgsMountRW       = "actualReadWrite"
	OsK8sVolumeNameMeta = "k8s_pvOrVolumeName"

	// Result status
	ResultStatusSuccess     = "Success"
	ResultStatusFailed      = "Failed"
	ResultStatusUnsupported = "Not supported"
	ResultMsgOpSuccess      = "Operation Success"

	// HTTP constants
	RespStatus200 = "200 OK"
	RespStatus201 = "201 Created"
	RespStatus202 = "202 Accepted"
	RespStatus401 = "401 Unauthorized"
	RespStatus500 = "500 Internal Server Error"

	HypTypeLibvirt = "libvirt"
	HypTypeKVM     = "kvm"
	HypTypePhyp    = "phyp"
	HypTypePvm     = "powervm"
	HypTypePvmKVM  = "novalink-kvm"

	StorageHostTypeGPFS = "gpfs"
	StorageHostTypeXIV  = "xiv"

	FlexPluginVendor    = "ibm"
	ScsiPath            = "/sys/class/scsi_host/"
	AttachedVolumeDir   = "/dev/disk/by-id/"
	DirNamePVMVIOS      = "wwn-0x"
	DirNamePrefixPVMLIO = "wwn-0x6001405"
	DirNamePrefixPVMXIV = "scsi-2"
	DirNamePrefixKVM    = "scsi-0QEMU_QEMU_HARDDISK_"
	PathPVMVIOS         = AttachedVolumeDir + DirNamePVMVIOS

	CMDSudo                = "/usr/bin/sudo"
	CMDLsBlk               = "/bin/lsblk"
	CMDMkDir               = "/bin/mkdir"
	CMDMkFS                = "/sbin/mkfs."
	CMDMount               = "/bin/mount"
	CMDUnmount             = "/bin/umount"
	CMDGrep                = "/bin/grep"
	CMDMultipath           = "/usr/sbin/multipath"
	CMDUdevAdm             = "/sbin/udevadm"
	CMDUdevAdmParamSettle  = "settle"
	CMDUdevAdmParamTrigger = "trigger"

	OSUser          = "OS_USERNAME"
	OSPassword      = "OS_PASSWORD"
	OSUserDomain    = "OS_USER_DOMAIN_NAME"
	OSProjectDomain = "OS_PROJECT_DOMAIN_NAME"
	OSProjectName   = "OS_TENANT_NAME"
	OSProjectID     = "OS_TENANT_ID"
	OSAuthURL       = "OS_AUTH_URL"
	OSCACert        = "OS_CACERT"

	URIProjects = "/v3/projects"

	MaxAttemptsToFindVolume = 24
	MaxAttemptsToTryLock    = 24
	ScsiScanLock            = "power-openstack-k8s-scsiscan.lck"
)

// FSTYPES : All linux file systems
var FSTYPES = []string{"ext2", "ext3", "ext4", "jfs", "ReiserFS", "XFS", "Btrfs"}

var FlexPluginDriver, FlexPluginVendorDriver, ProvisionerNameOnly, ProvisionerName, GlobalMountsDir string

// Utility to allow the caller to initialize the FlexVolume driver and provisioner to use a different naming scheme
func UpdateDriverPrefix(prefix string) {
	FlexPluginK8sDir := "/var/lib/kubelet/plugins"
	FlexPluginOcpDir := "/var/lib/origin/openshift.local.volumes/plugins"
	FlexPluginDriver = fmt.Sprintf("%s-volume-flex", prefix)
	FlexPluginVendorDriver = FlexPluginVendor + "/" + FlexPluginDriver
	ProvisionerNameOnly = fmt.Sprintf("%s-volume-provisioner", prefix)
	ProvisionerName = FlexPluginVendor + "/" + ProvisionerNameOnly
	// If the generic Kubernetes Directory doesn't exist, check for other possible legacy directories
	if _, err := os.Stat(FlexPluginK8sDir); os.IsNotExist(err) {
		if _, err = os.Stat(FlexPluginOcpDir); !os.IsNotExist(err) {
			FlexPluginK8sDir = FlexPluginOcpDir
		}
	} else {
		// ICP can now be configured on OCP. In that case use OCP global mounts directory as pods will use these now
		if _, err := os.Stat(FlexPluginOcpDir); !os.IsNotExist(err) {
			FlexPluginK8sDir = FlexPluginOcpDir
		}
	}
	GlobalMountsDir = fmt.Sprintf("%s/kubernetes.io/flexvolume/%s/%s/mounts/", FlexPluginK8sDir, FlexPluginVendor, FlexPluginDriver)
}

// We want to initialize the prefix to a generic name so that it can be used without being set
func init() {
	UpdateDriverPrefix("power-openstack-k8s")
}
