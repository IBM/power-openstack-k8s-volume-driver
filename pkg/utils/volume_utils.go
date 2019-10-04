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
	if dirs, err := ioutil.ReadDir(sysPath); err == nil {
		for _, dir := range dirs {
			dirName := dir.Name()
			// Check if its device mapper parent device
			if strings.HasPrefix(dirName, "dm-") {
				// Check if volume device is its slave
				expectedSlavePath := sysPath + dirName + "/slaves/" + deviceName
				if _, err := os.Lstat(expectedSlavePath); err == nil {
					dmPath := "/dev/" + dirName
					log.Debugf("DM Parent : %s", dmPath)
					return dmPath
				}
			}
		}
	}
	return devicePath
}

// HasFSOnVolumeDevice : Determines the File System installed
// on attached volume
func HasFSOnVolumeDevice(volPath string) (bool, error) {
	cmdStrs := []string{resources.CMDLsBlk, volPath}
	cmdStrs = append(cmdStrs, "--noheadings", "-o", "FSTYPE", "-f")
	cmdOutput, cmdError, err := RunCommand(resources.CMDSudo, cmdStrs)
	if err != nil {
		log.Errorf("Error running %s", cmdStrs)
		log.Error(cmdError)
		return false, err
	}
	cmdOutput = strings.TrimSpace(cmdOutput)
	// Check that the output has existence of a linux fliesystem
	if cmdOutput != "" {
		cmdOutput = strings.ToLower(cmdOutput)
		for _, fsType := range resources.FSTYPES {
			if strings.Contains(cmdOutput, strings.ToLower(fsType)) {
				log.Debugf("Attached Volume has FS %s", cmdOutput)
				return true, nil
			}
		}
	}
	// If we have not returned yet, it means that attached volume does not
	// have a file system.
	log.Debugf("Attached Volume does not has a file system %s", cmdOutput)
	return false, nil
}

// GetDeviceOfMount Function to get the block device or multipath device of mounted directory
func GetDeviceOfMount(volMountDir string) (string, error) {
	// Run mount | grep -w mountDirectory
	// cmdStrs := []string{fmt.Sprintf("%s | %s -w %s", resources.CMDMount, resources.CMDGrep, volMountDir)}
	// res, _, err := RunCommand(resources.CMDSudo, cmdStrs)
	res, err := RunPipedCommands(
		resources.CMDSudo, []string{resources.CMDMount},
		resources.CMDGrep, []string{"-w", volMountDir})
	if err != "" {
		return "", fmt.Errorf(err)
	}
	Log.Debugf("Mount check: %s", res)
	// The result is something of this format
	// /dev/mapper/mpathi on mountDirectory type ext4 (rw,relatime,seclabel,data=ordered)
	// Validate that we have only one line and six tokens as part of it.
	lines := strings.Split(string(res[:]), "\n")
	if len(lines) != 2 {
		Log.Errorf("Multiple mount points found %s", lines)
		return "", fmt.Errorf("Directory is not a mount point")
	}
	fields := strings.Fields(lines[0])
	if len(fields) != 6 {
		Log.Errorf("Error tokening mount output %s", fields)
		return "", fmt.Errorf("Error formatting mount output %d %s", len(fields), fields)
	}
	// The device is the first token
	device := fields[0]
	Log.Debugf("Device mounted on the %s directory is %s", volMountDir, device)
	return device, nil
}

// GetAssociatedBlockDevices : Function to get associated block device
// for a multipath device.
// Returns device mapper parent and associated block devices names
func GetAssociatedBlockDevices(devicePath string) (string, []string, error) {
	// Check if path is something valid
	if devicePath == "" || !strings.Contains(devicePath, "/dev") {
		errMsg := fmt.Sprintf("Device path is not valid: %s", devicePath)
		return "", nil, fmt.Errorf(errMsg)
	}
	// Check if the device is a multipath device
	if strings.Contains(devicePath, "/dev/mapper/mpath") {
		// Run multipath -l devicePath
		cmdStrs := []string{resources.CMDMultipath, "-l", devicePath}
		res, _, err := RunCommand(resources.CMDSudo, cmdStrs)
		if err != nil {
			Log.Errorf("Error running multipath command %s", err)
			return "", nil, err
		}
		line := strings.Split(string(res[:]), "\n")[0]
		// Check if the output says that this is not a valid multipath device
		if strings.Contains(line, "not a valid argument") {
			errMsg := fmt.Sprintf("Device does not seem to be multipath device: %s", line)
			Log.Error(errMsg)
			return "", nil, fmt.Errorf(errMsg)
		}
		// Split the line to find multipath parent. Ex it dm-3 below
		// mpathi (xxxxxx) dm-3 AIX     ,VDASD
		fields := strings.Fields(line)
		if len(fields) != 5 {
			return "", nil, fmt.Errorf("Error formatting multipath device output")
		}
		dmParent := fmt.Sprintf("/dev/%s", fields[2])
		Log.Debugf("Found device mapper parent to be %s", dmParent)
		if _, err := os.Stat(dmParent); os.IsNotExist(err) {
			return "", nil, fmt.Errorf("Can not find device mapper parent %s", dmParent)
		}
		// Now that we know the device mapper parent, lets find the associated slave devices
		blockDevices := FindSlaveDevicesOfMultipathParent(dmParent)
		Log.Debugf("Associated block devices are %s", blockDevices)
		return dmParent, blockDevices, nil
	}
	// If its not a multipath device, then return original device
	Log.Debugf("Not a multipath device: %s", devicePath)
	return "", []string{devicePath}, nil
}

// FindSlaveDevicesOfMultipathParent : Find slaves of device mapper parent
func FindSlaveDevicesOfMultipathParent(dm string) []string {
	var devices []string
	parts := strings.Split(dm, "/")
	if len(parts) != 3 || !strings.HasPrefix(parts[1], "dev") {
		return devices
	}
	dmParent := parts[2]
	// Slaves are found in /sys/block/dm/slaves
	slavesPath := filepath.Join("/sys/block/", dmParent, "/slaves/")
	if files, err := ioutil.ReadDir(slavesPath); err == nil {
		for _, f := range files {
			devices = append(devices, filepath.Join("/dev/", f.Name()))
		}
	}
	return devices
}

// RemoveMultipathForDevice : Cleans up given multipath
func RemoveMultipathForDevice(devicePath string) error {
	multipathName := devicePath
	if strings.Contains(devicePath, "/dev/mapper") {
		multipathName = strings.Split(devicePath, "/")[2]
	}
	cmdStrs := []string{resources.CMDMultipath, "-f", multipathName}
	_, _, err := RunCommand(resources.CMDSudo, cmdStrs)
	if err != nil {
		Log.Errorf("Error clearing multipath:  %s", err)
		return err
	}
	Log.Debugf("Multipath entry for %s removed", devicePath)
	return nil
}

// DeleteScsiDevice : Remove block device from SCSI sub-system
func DeleteScsiDevice(devicePath string) {
	pathArr := strings.Split(devicePath, "/")
	deviceName := pathArr[len(pathArr)-1]
	fileName := "/sys/block/" + deviceName + "/device/delete"
	data := []byte("1")
	ioutil.WriteFile(fileName, data, 0666)
	Log.Infof("Deleted device from SCSI system: %s", fileName)
}

// ScsiHostScan : Function rescans the scsi bus
func ScsiHostScan() {
	scsiPath := resources.ScsiPath
	if dirs, err := ioutil.ReadDir(scsiPath); err == nil {
		for _, f := range dirs {
			name := scsiPath + f.Name() + "/scan"
			data := []byte("- - -")
			err := ioutil.WriteFile(name, data, 0666)
			if err != nil {
				log.Warningf("Could not rescan file %s", name)
			}
			Log.Debugf("Scsi scan done for file %s", f.Name())
		}
		Log.Debug("Scsi scan done")
	}
}

// UdevdHandleEvents : Indicate udevd to handle device creation and deletion events
func UdevdHandleEvents(volPath string) error {
	cmdStrs := []string{resources.CMDUdevAdm, resources.CMDUdevAdmParamSettle}
	_, _, err := RunCommand(resources.CMDSudo, cmdStrs)
	if err != nil {
		log.Errorf("Error running %s", cmdStrs)
		return err
	}
	log.Debug("Ran command udevadm settle")
	cmdStrs = []string{resources.CMDUdevAdm, resources.CMDUdevAdmParamTrigger}
	// If the directory of attached volume exists, we should run trigger only for that path
	if _, err = os.Stat(volPath); !os.IsNotExist(err) {
		log.Debugf("Found directory for attached volume %s", volPath)
		cmdStrs = []string{resources.CMDUdevAdm, resources.CMDUdevAdmParamTrigger, volPath}
	}
	_, _, err = RunCommand(resources.CMDSudo, cmdStrs)
	if err != nil {
		log.Errorf("Error running %s", cmdStrs)
		return err
	}
	log.Debugf("Ran command udevadm trigger")
	return nil
}
