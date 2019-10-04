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
	volumes_v3 "github.com/gophercloud/gophercloud/openstack/blockstorage/v3/volumes"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
)

/********************** Structure definitions ************************/

// Response structure that defines attributes of expected response
type Response struct {
	// Response status
	Status string `json:"status"`
	// Response message
	Message string `json:"message,omitempty"`
	// Device name to be returned as part of attach operation
	DeviceName string `json:"device,omitempty"`
	// Volume name to be returned as part of getvolumename operation
	VolumeName string `json:"volumeName,omitempty"`
	// Attached device name to be returned as part of attach operation
	Attached *bool `json:"attached,omitempty"`
	// List of capabilities to be returned as part of init operation
	Capabilities map[string]bool `json:"capabilities,omitempty"`
}

// StorageRegistration : structure representing the registration information of storage host
type StorageRegistration struct {
	HostName    string `json:"host_display_name"`
	AccessIP    string `json:"access_ip"`
	HostType    string `json:"host_type"`
	DriverType  string `json:"driver_volume_type"`
	AccessState string `json:"access_state"`
}

// Hypervisor : Structure representing hypervisor
type Hypervisor struct {

	// Status of the hypervisor, either "enabled" or "disabled".
	Status string `json:"status"`

	// State of the hypervisor, either "up" or "down".
	State string `json:"state"`

	// HostIP is the hypervisor's IP address.
	HostIP string `json:"host_ip"`

	// FreeRAMMB is the free RAM in the hypervisor, measured in MB.
	FreeRamMB int `json:"free_ram_mb"`

	// HypervisorHostname is the hostname of the hypervisor.
	HypervisorHostname string `json:"hypervisor_hostname"`

	// HypervisorType is the type of hypervisor.
	HypervisorType string `json:"hypervisor_type"`

	// HypervisorVersion is the version of the hypervisor.
	HypervisorVersion int `json:"-"`

	// ID is the unique ID of the hypervisor.
	ID int `json:"id"`

	// MemoryMB is the total memory of the hypervisor, measured in MB.
	MemoryMB int `json:"memory_mb"`

	// MemoryMBUsed is the used memory of the hypervisor, measured in MB.
	MemoryMBUsed int `json:"memory_mb_used"`

	// RunningVMs is the The number of running vms on the hypervisor.
	RunningVMs int `json:"running_vms"`

	// VCPUs is the total number of vcpus on the hypervisor.
	VCPUs int `json:"vcpus"`

	// VCPUsUsed is the number of used vcpus on the hypervisor.
	VCPUsUsed int `json:"vcpus_used"`
}

// OSServerAttrsExt : Extension to base Server object
type OSServerAttrsExt struct {
	HypervisorHostname string `json:"OS-EXT-SRV-ATTR:hypervisor_hostname"`
}

// OSServer : Extend gophercloud Server to get VM's host as part of result
type OSServer struct {
	servers.Server
	OSServerAttrsExt
}

// OSVolumeAttrsExt : Extension to base Volume object
type OSVolumeAttrsExt struct {
	BackendHost string `json:"backend_host"`
}

// OSVolume : Extend gophercloud Volume to get Volume's Storage host
type OSVolume struct {
	volumes_v3.Volume
	OSVolumeAttrsExt
}

/********************** Structure definitions end ************************/
