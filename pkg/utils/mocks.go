/*
  Copyright IBM Corp. 2018.

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
package util

import (
	resources "github.com/IBM/power-openstack-k8s-volume-driver/pkg/resources"
	"github.com/gophercloud/gophercloud"
	volumes_v3 "github.com/gophercloud/gophercloud/openstack/blockstorage/v3/volumes"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/hypervisors"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
)

/*
The OpenstackCloudMock is structured as follows:
Hosts:
host_1 -- PowerVM
host_2 -- KVM

VMs:
vm_1 : { IP: 1.2.3.4 , volume: vol_1, volume_wwn: wwn_1, backend_host: svc, host: host_1}
vm_2 : { IP: 1.2.3.5 , volume: vol_2, volume_wwn: wwn_2, backend_host: gpfs, host: host_2 }
vm_3 : { IP: 1.2.3.6 , volume: vol_3, volume_wwn: wwn_3, backend_host: xiv, host: host_1 }
vm_4 : { IP: 1.2.3.7 , volume: vol_4, volume_wwn: wwn_4, backend_host: gpfs, host: host_1 }
vm_5 : { IP: 1.2.3.8 , volume: vol_5, volume_wwn: wwn_5, backend_host: generic, host: host_1 }

VM1 represents PowerVM VIOS
VM2 represents KVM virtio-scsi
VM3 represents PowerVM VIOS for XIV
VM4 represents PowerVM LIO
VM5 represents Openstack VM on PowerVM
*/

// OpenstackCloudMock : Our mock that we will plug in for tests.
type OpenstackCloudMock struct{}

/*****  Implement OpenstackCloudI interface methods  *****/

// GetProviderClient : Returns back the embedded ProviderClient in the Interface
func (opnStk *OpenstackCloudMock) GetProviderClient() *gophercloud.ProviderClient {
	return nil
}

// AttachVolumeToVM :
func (opnStk *OpenstackCloudMock) AttachVolumeToVM(vmID string, volumeID string, volume *resources.OSVolume) (bool, error) {
	if vmID == "vm_1" && volumeID == "vol_1" {
		return true, nil
	}
	return false, nil
}

// DetachVolumeFromVM :
func (opnStk *OpenstackCloudMock) DetachVolumeFromVM(vmID string, volumeID string, volume *resources.OSVolume) (bool, error) {
	return true, nil
}

// IsVolumeAttached :
func (opnStk *OpenstackCloudMock) IsVolumeAttached(vmID string, volumeID string) (bool, error) {
	if vmID == "vm_1" && volumeID == "vol_1" {
		return true, nil
	}
	return false, nil
}

// GetVolumeByMetadataProperty :
func (opnStk *OpenstackCloudMock) GetVolumeByMetadataProperty(
	volumeMeta map[string]string) (*[]resources.OSVolume, error) {
	volID := volumeMeta[resources.OsK8sVolumeNameMeta]
	vol, _ := GetOSVolumeByID(opnStk, volID)
	vols := []resources.OSVolume{*vol}
	return &vols, nil
}

// UpdateVolumeMetadata :
func (opnStk *OpenstackCloudMock) UpdateVolumeMetadata(volumeID string, volumeMeta map[string]string, isDelete bool) error {
	return nil
}

// GetServerIDFromNodeName :
func (opnStk *OpenstackCloudMock) GetServerIDFromNodeName(nodeName string) (string, error) {
	if nodeName == "1.2.3.4" {
		return "vm_1", nil
	} else if nodeName == "1.2.3.5" {
		return "vm_2", nil
	} else if nodeName == "1.2.3.6" {
		return "vm_3", nil
	} else if nodeName == "1.2.3.7" {
		return "vm_4", nil
	} else if nodeName == "1.2.3.8" {
		return "vm_5", nil
	}
	return "", nil
}

// ListHypervisors :
func (opnStk *OpenstackCloudMock) ListHypervisors() (*[]hypervisors.Hypervisor, error) {
	hyp1 := hypervisors.Hypervisor{
		ID:                 1,
		HypervisorHostname: "host_1",
		HypervisorType:     "powervm",
	}
	hyp2 := hypervisors.Hypervisor{
		ID:                 1,
		HypervisorHostname: "host_2",
		HypervisorType:     "libvirt",
	}
	hyps := []hypervisors.Hypervisor{hyp1, hyp2}
	return &hyps, nil
}

// GetAllOSVMs :
func (opnStk *OpenstackCloudMock) GetAllOSVMs() (*[]resources.OSServer, error) {
	//var nwAddrs interface{}
	nwAddrs1 := map[string]interface{}{"addr": "1.2.3.4"}
	nwAddrs2 := map[string]interface{}{"addr": "1.2.3.5"}
	nwAddrs3 := map[string]interface{}{"addr": "1.2.3.6"}
	nwAddrs4 := map[string]interface{}{"addr": "1.2.3.7"}
	nwAddrs5 := map[string]interface{}{"addr": "1.2.3.8"}

	vm1 := resources.OSServer{
		Server: servers.Server{
			ID:        "vm_1",
			Addresses: map[string]interface{}{"network_2230": []interface{}{nwAddrs1}},
		},
		OSServerAttrsExt: resources.OSServerAttrsExt{HypervisorHostname: "host_1"},
	}
	vm2 := resources.OSServer{
		Server: servers.Server{
			ID:        "vm_2",
			Addresses: map[string]interface{}{"network_2230": []interface{}{nwAddrs2}},
		},
		OSServerAttrsExt: resources.OSServerAttrsExt{HypervisorHostname: "host_2"},
	}
	vm3 := resources.OSServer{
		Server: servers.Server{
			ID:        "vm_3",
			Addresses: map[string]interface{}{"network_2230": []interface{}{nwAddrs3}},
		},
		OSServerAttrsExt: resources.OSServerAttrsExt{HypervisorHostname: "host_1"},
	}
	vm4 := resources.OSServer{
		Server: servers.Server{
			ID:        "vm_4",
			Addresses: map[string]interface{}{"network_2230": []interface{}{nwAddrs4}},
		},
		OSServerAttrsExt: resources.OSServerAttrsExt{HypervisorHostname: "host_1"},
	}
	vm5 := resources.OSServer{
		Server: servers.Server{
			ID:        "vm_5",
			Addresses: map[string]interface{}{"network_2230": []interface{}{nwAddrs5}},
		},
		OSServerAttrsExt: resources.OSServerAttrsExt{HypervisorHostname: "host_1"},
	}
	vms := []resources.OSServer{vm1, vm2, vm3, vm4, vm5}
	return &vms, nil
}

// GetOSVolumeByID :
func (opnStk *OpenstackCloudMock) GetOSVolumeByID(volumeID string) (*resources.OSVolume, error) {
	var osVol resources.OSVolume
	if volumeID == "vol_1" {
		volAttachment := volumes_v3.Attachment{ID: "1", ServerID: "vm_1", VolumeID: "vol_1"}
		vol := volumes_v3.Volume{
			ID:          "05a0a13c-f839-4c67-bd02-28b6fb6fada3",
			Metadata:    map[string]string{"volume_wwn": "wwn_1"},
			Attachments: []volumes_v3.Attachment{volAttachment},
		}
		attrs := resources.OSVolumeAttrsExt{
			BackendHost: "svc",
		}
		osVol = resources.OSVolume{Volume: vol, OSVolumeAttrsExt: attrs}
	} else if volumeID == "vol_2" {
		volAttachment := volumes_v3.Attachment{ID: "2", ServerID: "vm_2", VolumeID: "vol_2"}
		vol := volumes_v3.Volume{
			ID:          "9ddd4949-5117-4ad8-82b2-8ab37b690078",
			Metadata:    map[string]string{"volume_wwn": "wwn_2"},
			Attachments: []volumes_v3.Attachment{volAttachment},
		}
		attrs := resources.OSVolumeAttrsExt{
			BackendHost: "gpfs_host",
		}
		osVol = resources.OSVolume{Volume: vol, OSVolumeAttrsExt: attrs}
	} else if volumeID == "vol_3" {
		volAttachment := volumes_v3.Attachment{ID: "3", ServerID: "vm_3", VolumeID: "vol_3"}
		vol := volumes_v3.Volume{
			ID:          "9ddd4949-5117-4ad8-82b2-8ab37b690079",
			Metadata:    map[string]string{"volume_wwn": "wwn_3"},
			Attachments: []volumes_v3.Attachment{volAttachment},
		}
		attrs := resources.OSVolumeAttrsExt{
			BackendHost: "xiv_host",
		}
		osVol = resources.OSVolume{Volume: vol, OSVolumeAttrsExt: attrs}
	} else if volumeID == "vol_4" {
		volAttachment := volumes_v3.Attachment{ID: "4", ServerID: "vm_4", VolumeID: "vol_4"}
		vol := volumes_v3.Volume{
			ID:          "9ddd4949-5117-4ad8-82b2-8ab37b690080",
			Metadata:    map[string]string{"volume_wwn": "wwn_4"},
			Attachments: []volumes_v3.Attachment{volAttachment},
		}
		attrs := resources.OSVolumeAttrsExt{
			BackendHost: "gpfs_host",
		}
		osVol = resources.OSVolume{Volume: vol, OSVolumeAttrsExt: attrs}
	} else if volumeID == "vol_5" {
		volAttachment := volumes_v3.Attachment{ID: "5", ServerID: "vm_5", VolumeID: "vol_5"}
		vol := volumes_v3.Volume{
			ID:          "9ddd4949-5117-4ad8-82b2-8ab37b690081",
			Metadata:    map[string]string{"volume_wwn": "wwn_5"},
			Attachments: []volumes_v3.Attachment{volAttachment},
		}
		attrs := resources.OSVolumeAttrsExt{
			BackendHost: "generic",
		}
		osVol = resources.OSVolume{Volume: vol, OSVolumeAttrsExt: attrs}
	}

	return &osVol, nil
}

// GetStorageHostRegistration :
func (opnStk *OpenstackCloudMock) GetStorageHostRegistration(hostname string) (*resources.StorageRegistration, error) {
	regData := resources.StorageRegistration{}
	if hostname == "svc" {
		regData.HostType = "svc"
	} else if hostname == "gpfs_host" {
		regData.HostType = "gpfs"
	} else if hostname == "xiv_host" {
		regData.HostType = "xiv"
	} else if hostname == "generic" {
		return nil, nil
	}
	return &regData, nil
}
