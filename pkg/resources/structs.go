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
