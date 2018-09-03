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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	resources "github.com/IBM/power-openstack-k8s-volume-driver/pkg/resources"
)

// GetVolumeDirectoryName : Given VM IP and volume ID, this function determines the directory name
// on the VM where the volume will show up after SCSI rescan.
func GetVolumeDirectoryName(cloud OpenstackCloudI, nodeAddr string, volumeID string, volume *resources.OSVolume) (string, error) {
	var directoryName string
	vmMap, err := CreateVMIPToVMDetailsMap(cloud)
	if err != nil {
		return "", err
	}

	hostMap, err := CreateHostnameToDetailsMap(cloud)
	if err != nil {
		return "", err
	}

	if volume == nil {
		volume, err = cloud.GetOSVolumeByID(volumeID)
		if err != nil {
			return "", err
		}
	}

	if vm, ok := vmMap[nodeAddr]; ok {
		vmHostName := vm.HypervisorHostname
		if host, ok := hostMap[vmHostName]; ok {
			hypType := host.HypervisorType
			if hypType == resources.HypTypeLibvirt || hypType == resources.HypTypeKVM || hypType == resources.HypTypePvmKVM {
				log.Debug("Looking for directory name for KVM")
				directoryName = getDirectoryNameKVM(volume)
			} else if hypType == resources.HypTypePhyp || hypType == resources.HypTypePvm {
				// Get volume's storage host
				volHost := volume.BackendHost
				regData, err := cloud.GetStorageHostRegistration(volHost)
				if err != nil {
					return "", err
				}
				if regData != nil {
					if regData.HostType == resources.StorageHostTypeGPFS {
						// PowerVM LIO
						log.Debug("Looking for directory name for PowerVM LIO")
						directoryName = getDirectoryNamePvmLIO(volume)
					} else if regData.HostType == resources.StorageHostTypeXIV {
						// Handle XIV
						log.Debug("Looking for directory name for PowerVM VIOS for XIV storage")
						directoryName = getDirectoryNamePvmXiv(volume)
					} else {
						// PowerVM VIOS
						log.Debug("Looking for directory name for PowerVM VIOS")
						directoryName = getDirectoryNameForPhyp(volume)
					}
				} else {
					// Default to PowerVM VIOS
					log.Debug("No registration data. Looking for directory name for PowerVM VIOS")
					directoryName = getDirectoryNameForPhyp(volume)
				}
			}
		}
	}
	return directoryName, nil
}

// GetDirectoryNameForPhyp : For SVC backed volume, this method returns
// the directory path where the volume will show up as a directory on the VM
func getDirectoryNameForPhyp(vol *resources.OSVolume) string {
	wwn := vol.Metadata["volume_wwn"]
	volPath := getDirectoryNameForPhypByWWN(wwn)
	return volPath
}

// GetDirectoryNameForPhypByWWN : Given Volume's WWN, this method returns
// the directory path where the volume will show up as a directory on the VM
func getDirectoryNameForPhypByWWN(wwn string) string {
	volPath := string(resources.PathPVMVIOS)
	if wwn != "" {
		volPath += strings.ToLower(wwn)
	}
	Log.Debugf("Expected path of volume is %s", volPath)
	return volPath
}

func getDirectoryNamePvmLIO(vol *resources.OSVolume) string {
	volPath := string(resources.AttachedVolumeDir)
	volumeID := vol.ID
	volPath += fmt.Sprintf(resources.DirNamePrefixPVMLIO+"%s", strings.Replace(volumeID, "-", "", -1)[:25])
	Log.Debugf("Expected path of volume is %s", volPath)
	return volPath
}

func getDirectoryNamePvmXiv(vol *resources.OSVolume) string {
	volPath := string(resources.AttachedVolumeDir)
	wwn := vol.Metadata["volume_wwn"]
	if wwn != "" {
		volPath += resources.DirNamePrefixPVMXIV + strings.ToLower(wwn)
	}
	Log.Debugf("Expected path of volume is %s", volPath)
	return volPath
}

func getDirectoryNameKVM(vol *resources.OSVolume) string {
	volPath := string(resources.AttachedVolumeDir)
	volumeID := vol.ID
	volPath += fmt.Sprintf(resources.DirNamePrefixKVM+"%s", volumeID[:20])
	Log.Debugf("Expected path of volume is %s", volPath)
	return volPath
}

// FindAttachedVolumeDirectoryPath : Returns directory path of attached volume
// For multipath device, return device mapper parent
func FindAttachedVolumeDirectoryPath(dirName string) string {
	devicePath, err := filepath.EvalSymlinks(dirName)
	if err != nil {
		return ""
	}
	log.Debugf("Device path is %s \n", devicePath)
	devicePathParts := strings.Split(devicePath, "/")
	deviceName := devicePathParts[len(devicePathParts)-1]

	// Iterate over the block devices
	sysPath := "/sys/block/"
	if dirs, err := ioutil.ReadDir(sysPath); err != nil {
		for _, dir := range dirs {
			dirName := dir.Name()
			// Check if its device mapper parent device
			if strings.HasPrefix(dirName, "dm-") {
				// Check if volume device is its slave
				expectedSlavePath := sysPath + dirName + "/slaves/" + deviceName
				if _, err := os.Lstat(expectedSlavePath); err != nil {
					dmPath := "/dev/" + dirName
					log.Debugf("DM Parent : %s", dmPath)
					return dmPath
				}
			}
		}
	}
	return devicePath
}
