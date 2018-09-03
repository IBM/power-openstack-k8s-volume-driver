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
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	resources "github.com/IBM/power-openstack-k8s-volume-driver/pkg/resources"
	utils "github.com/IBM/power-openstack-k8s-volume-driver/pkg/utils"
)

// Map to hold the operation to its allowed operation params
var operationArgsMap = make(map[string]map[string][]string)

// Global logger
var log = utils.Log

// Reference to openstack
var cloud utils.OpenstackCloudI

/********************** Driver operations ************************/

// Implements <driver> init API
func driverInit() map[string]bool {
	log.Info("\n Init called")
	details := map[string]bool{"attach": true}
	return details
}

// Implements <driver> getvolumename <json_params> API
func getVolumeName(jsonArgs map[string]string) map[string]string {
	log.Infof("\n getVolumeName called with %s", jsonArgs)
	volName := jsonArgs[resources.OsArgsVolID]
	details := map[string]string{
		"status":  resources.ResultStatusSuccess,
		"msg":     resources.ResultMsgOpSuccess,
		"volName": volName,
	}
	return details
}

// Implements <driver> isattached nodename <json_params> API
func isAttached(nodeName string, jsonArgs map[string]string) map[string]string {
	log.Infof("\n isAttached called with %s %s", nodeName, jsonArgs)
	details := make(map[string]string)
	// Get openstack VM and volume ID
	volumeID := jsonArgs[resources.OsArgsVolID]
	vmID, err := utils.GetVMID(cloud, nodeName)
	if err != nil {
		return utils.ErrorStruct(fmt.Sprintf("Could not find VM with id %s. Error is %s", vmID, err))
	}
	// Check if volume is attached to VM
	found, err := utils.IsVolumeAttached(cloud, vmID, volumeID)
	if err != nil {
		return utils.ErrorStruct(fmt.Sprintf("Unable to determine if volume is attached to VM. Error is %s", err))
	} else if found {
		log.Debugf("Found volume %s attached to VM", volumeID)
		details = map[string]string{
			"status":   resources.ResultStatusSuccess,
			"msg":      resources.ResultMsgOpSuccess,
			"attached": "true",
		}
	} else {
		log.Debugf("Could not find volume %s attached to VM", volumeID)
		details = map[string]string{
			"status":   resources.ResultStatusSuccess,
			"msg":      resources.ResultMsgOpSuccess,
			"attached": "false",
		}
	}
	return details
}

// Implements <driver> attach <json_params> nodename API
func attach(nodeName string, jsonArgs map[string]string) map[string]string {
	log.Infof("\n attach called with %s %s", nodeName, jsonArgs)

	// Extract volume id and volume name from jsonParams
	volumeID := jsonArgs[resources.OsArgsVolID]
	volumeName := jsonArgs[resources.K8sArgPV]

	// Get VM id
	vmID, err := utils.GetVMID(cloud, nodeName)
	if err != nil {
		return utils.ErrorStruct(fmt.Sprintf("Could not find VM with id %s. Error is %s", vmID, err))
	}

	// Get volume
	volume, err := utils.GetOSVolumeByID(cloud, volumeID)
	if err != nil {
		return utils.ErrorStruct(fmt.Sprintf("Could not find volume with id %s. Error is %s", volumeID, err.Error()))
	}

	// Update volume metadata to store the volume name. This is required to identify the
	// volume in detach API as in detach, only volume name is provided.
	volumeMeta := make(map[string]string)
	// Add the K8s volume name to metadata
	volumeMeta[resources.OsK8sVolumeNameMeta] = volumeName
	// Call openstack API to update
	err = utils.UpdateVolumeMetadata(cloud, volumeID, volumeMeta, false)
	if err != nil {
		return utils.ErrorStruct(fmt.Sprintf("Could not update volume %s metadata to store "+
			"kubernetes volume name. Error is %s", volumeID, err.Error()))
	}

	// Attach volume to VM. Pass volume to avoid making the get volume call in the method again.
	isSuccess, err := utils.AttachVolumeToVM(cloud, vmID, volumeID, volume)
	if err != nil {
		return utils.ErrorStruct(fmt.Sprintf("Could not attach volume %s to VM %s. Error is %s", volumeID, vmID, err))
	} else if !isSuccess {
		return utils.ErrorStruct(fmt.Sprintf("Could not attach volume %s to VM %s.", volumeID, vmID))
	}

	// Find the path of the directory where volume will show up on VM
	volPath, err := utils.GetVolumeDirectoryName(cloud, utils.ResolveNodeAddress(nodeName), volumeID, volume)
	if err != nil {
		return utils.ErrorStruct(fmt.Sprintf("Could not determine volume directory name. Error is %s", err))
	}
	log.Infof("Returning the expected device path as %s", volPath)
	return map[string]string{
		"status":     resources.ResultStatusSuccess,
		"msg":        resources.ResultMsgOpSuccess,
		"deviceName": volPath,
	}
}

// Implements <driver> waitforattach device_path <json_params> API
func waitForAttach(devicePath string, jsonArgs map[string]string) map[string]string {
	log.Infof("\n waitForAttach called with %s %s", devicePath, jsonArgs)
	var volPath string

	// Get volume id from json params
	volumeID := jsonArgs[resources.OsArgsVolID]
	volPath = devicePath
	// Loop for max of 120 seconds to find the attached volume
	for i := 0; i < resources.MaxAttemptsToFindVolume; i++ {
		// Sleep for 5 seconds
		time.Sleep(5 * time.Second)
		// Run scsi scan to discover the volume directory on VM
		utils.ScsiHostScan()
		// Let udevd handle device events
		err := utils.UdevdHandleEvents()
		if err != nil {
			log.Warningf("There was error while at udevd. Error is %s", err)
		}
		// Check if directory is available now after scan
		if fileInfo, err := os.Lstat(volPath); err == nil {
			// Find the symbolic link to the file
			if fileInfo.Mode() != 0&os.ModeSymlink {
				volDevicePath := utils.FindAttachedVolumeDirectoryPath(volPath)
				// If the file was link and we failed to read the link, return failed status
				if volDevicePath == "" {
					log.Errorf("Error finding link %s", err)
					return utils.ErrorStruct(fmt.Sprintf("Could not find symbolic link of attached volume with id %s. Error is %s", volumeID, err))
				}

				log.Debugf("Found directory of attached volume %s", volDevicePath)
				return map[string]string{
					"status":     resources.ResultStatusSuccess,
					"msg":        resources.ResultMsgOpSuccess,
					"deviceName": volDevicePath,
				}
			}
			break
		}
	}
	return utils.ErrorStruct(fmt.Sprintf("Could not find directory of attached volume with id %s", volumeID))
}

// Implements <driver> mountdevice mount_dir device_path <json_params> API
func mountDevice(mountPath string, devicePath string, jsonArgs map[string]string) map[string]string {
	log.Infof("\n mountDevice called with %s %s", mountPath, jsonArgs)
	fsType := jsonArgs[resources.K8sArgFSType]

	// Create File system on directory of attached volume
	if fsType == "" {
		// Assume default
		fsType = "ext4"
	}
	cmdStrs := []string{resources.CMDMkFS + fsType, devicePath}
	// We want to force the filesystem create if the command has the option, but very few actually do
	if strings.HasPrefix(fsType, "ext") || strings.HasPrefix(fsType, "ntfs") {
		cmdStrs = append(cmdStrs, "-F")
	}
	err := utils.RunCommand(resources.CMDSudo, cmdStrs)
	if err != nil {
		log.Errorf("Could not create file system on attached volume directory %s. Error is %s", devicePath, err)
		return utils.ErrorStruct(fmt.Sprintf("Could not create file system on attached volume directory %s. Error is %s", devicePath, err))
	}
	log.Debugf("Created %s file system at %s", fsType, devicePath)

	// Create mount directory as specified by mountPath
	cmdStrs = []string{resources.CMDMkDir, "-p", mountPath}
	err = utils.RunCommand(resources.CMDSudo, cmdStrs)
	if err != nil {
		log.Errorf("Could not create directory %s to mount attached volume", mountPath)
		return utils.ErrorStruct(fmt.Sprintf("Could not create directory %s to mount attached volume. Error is %s", mountPath, err))
	}
	log.Debugf("Created directory %s for mounting volume ", mountPath)

	// Mount the volume at mount path
	cmdStrs = []string{resources.CMDMount, devicePath, mountPath}
	// If they asked to mount it as read-only, add in the -r option here
	// Since the the kubernetes.io/readwrite argument isn't accurate currently,
	// we will also look at our own flag for now until the other one is fixed
	if jsonArgs[resources.K8sArgMountRW] == "ro" || jsonArgs[resources.OsArgsMountRW] == "ro" {
		cmdStrs = []string{resources.CMDMount, "-r", devicePath, mountPath}
	}
	err = utils.RunCommand(resources.CMDSudo, cmdStrs)
	if err != nil {
		log.Errorf("Could not mount %s directory to mount path %s", devicePath, mountPath)
		return utils.ErrorStruct(fmt.Sprintf("Could not mount %s directory to mount path %s. Error is %s", devicePath, mountPath, err))
	}

	log.Debugf("Mounted %s directory to mount path %s", devicePath, mountPath)
	return map[string]string{
		"status": resources.ResultStatusSuccess,
		"msg":    resources.ResultMsgOpSuccess,
	}
}

// Implements <driver> mount mount_dir <json_params> API'
func mount(mountDir string, jsonArgs map[string]string) map[string]string {
	log.Infof("\n mount called with %s %s", mountDir, jsonArgs)
	volumeName := jsonArgs[resources.K8sArgPV]
	volumeMountDir := resources.GlobalMountsDir + volumeName

	// Create pod mount directory
	cmdStrs := []string{resources.CMDMkDir, "-p", mountDir}
	err := utils.RunCommand(resources.CMDSudo, cmdStrs)
	if err != nil {
		log.Errorf("Could not create directory %s to mount attached volume", mountDir)
		return utils.ErrorStruct(fmt.Sprintf("Could not create directory %s to mount attached volume. Error is %s", mountDir, err))
	}

	// Bind mount
	cmdStrs = []string{resources.CMDMount, "--bind", volumeMountDir, mountDir}
	err = utils.RunCommand(resources.CMDSudo, cmdStrs)
	if err != nil {
		log.Errorf("Could not bind mount %s on %s. Error is %s", volumeMountDir, mountDir, err)
		return utils.ErrorStruct(fmt.Sprintf("Could not bind mount %s on %s. Error is %s", volumeMountDir, mountDir, err))
	}

	log.Debugf("Bind mounted %s %s", volumeMountDir, mountDir)
	return map[string]string{
		"status": resources.ResultStatusSuccess,
		"msg":    resources.ResultMsgOpSuccess,
	}
}

// Implements <driver> detach device_path nodename API
func detach(devicePath string, nodeName string) map[string]string {
	log.Infof("\n detach called with %s %s", devicePath, nodeName)
	// Detach is supposed to be called with device path, but it gets called with volume name
	// https://github.com/kubernetes/kubernetes/blob/master/pkg/volume/flexvolume/detacher.go#L44
	// so devicePath is really volume name

	// Retrieve volume id by querying its metadata for volumename
	volumeMeta := map[string]string{resources.OsK8sVolumeNameMeta: devicePath}
	vols, err := utils.GetVolumeByMetadataProperty(cloud, volumeMeta)
	if err != nil {
		return utils.ErrorStruct(fmt.Sprintf("Could not find volume with name %s. Error is %s", devicePath, err))
	}
	if len(*vols) > 1 {
		log.Errorf("Found more than one volume with volume name %s set in metadata.", devicePath)
		return utils.ErrorStruct(fmt.Sprintf("Found more than one volume with volume name %s set in metadata.", devicePath))
	}
	// Extract the volume ID
	volumeID := (*vols)[0].ID
	volume := (*vols)[0]
	// Get VM ID
	vmID, err := utils.GetVMID(cloud, nodeName)
	if err != nil {
		return utils.ErrorStruct(fmt.Sprintf("Could not find VM with id %s. Error is %s", vmID, err))
	}

	// Detach volume from VM. Pass volume so as to avoid making REST call to volume API again.
	isSuccess, err := utils.DetachVolumeFromVM(cloud, vmID, volumeID, volume)
	if err != nil {
		return utils.ErrorStruct(fmt.Sprintf("Could not detach volume from VM with id %s. Error is %s", vmID, err))
	} else if !isSuccess {
		return utils.ErrorStruct(fmt.Sprintf("Could not detach volume from VM with id %s.", vmID))
	}

	// We can just return now since we don't want to cleanup the metadata in-case it is a multi-attach scenario
	return map[string]string{
		"status": resources.ResultStatusSuccess,
		"msg":    resources.ResultMsgOpSuccess,
	}
}

// Implements <driver> waitfordetach device_path API
func waitForDetach(devicePath string) map[string]string {
	// Doesn't really gets called
	log.Infof("\n waitfordetach called with %s", devicePath)
	details := map[string]string{
		"status": resources.ResultStatusSuccess,
		"msg":    resources.ResultMsgOpSuccess,
	}
	return details
}

// Implements <driver> unmount_device mount_dir API
func unmountDevice(mountPath string) map[string]string {
	log.Infof("\n unmountDevice called with %s", mountPath)
	cmdStrs := []string{resources.CMDUnmount, mountPath}
	err := utils.RunCommand(resources.CMDSudo, cmdStrs)

	if err != nil {
		log.Errorf("Could not unmount volume directory %s. Error is %s", mountPath, err)
		return utils.ErrorStruct(fmt.Sprintf("Could not unmount volume directory %s. Error is %s", mountPath, err))
	}
	details := map[string]string{
		"status": resources.ResultStatusSuccess,
		"msg":    resources.ResultMsgOpSuccess,
	}
	return details
}

// Implements <driver> unmount mount_dir API
func unmount(mountDir string) map[string]string {
	log.Infof("\n unmount called with %s", mountDir)
	cmdStrs := []string{resources.CMDUnmount, mountDir}
	err := utils.RunCommand(resources.CMDSudo, cmdStrs)
	if err != nil {
		log.Errorf("Could not unmount volume directory %s. Error is %s", mountDir, err)
		return utils.ErrorStruct(fmt.Sprintf("Could not unmount volume directory %s. Error is %s", mountDir, err))
	}
	details := map[string]string{
		"status": resources.ResultStatusSuccess,
		"msg":    "Unmounted volume directory " + mountDir,
	}
	return details
}

/********************** Driver operations end ************************/

func initCloud() error {
	var err error
	cloud, err = utils.CreateOpenstackClient()
	if err != nil {
		return err
	}
	log.Info("Authenticated with openstack")
	return nil
}

func createRespMsg(opType string, args []string) resources.Response {
	var resp resources.Response
	var details map[string]string

	// Initialize openstack provider for attach/detach operations. Needed only on controller.
	if opType == resources.OpAttach || opType == resources.OpDetach || opType == resources.OpIsAttached {
		err := initCloud()
		if err != nil {
			return resources.Response{
				Status:  resources.ResultStatusFailed,
				Message: fmt.Sprintf("Could not authenticate with Openstack. Error is %s", err),
			}
		}
	}

	switch {
	case opType == resources.OpInit:
		respDetails := driverInit()
		resp = resources.Response{Status: resources.ResultStatusSuccess, Message: resources.ResultMsgOpSuccess, Capabilities: respDetails}

	case opType == resources.OpGetVolumeName:
		details = getVolumeName(utils.GetJSONArgs(args[1]))
		volName := details["volName"]
		resp = resources.Response{Status: details["status"], Message: details["msg"], VolumeName: volName}

	case opType == resources.OpIsAttached:
		details := isAttached(args[2], utils.GetJSONArgs(args[1]))
		isVolAttached, _ := strconv.ParseBool(details["attached"])
		resp = resources.Response{Status: details["status"], Message: details["msg"], Attached: &isVolAttached}

	case opType == resources.OpAttach:
		details := attach(args[2], utils.GetJSONArgs(args[1]))
		deviceName := details["deviceName"]
		resp = resources.Response{Status: details["status"], Message: details["msg"], DeviceName: deviceName}

	case opType == resources.OpWaitForAttach:
		details = waitForAttach(args[1], utils.GetJSONArgs(args[2]))
		deviceName := details["deviceName"]
		resp = resources.Response{Status: details["status"], Message: details["msg"], DeviceName: deviceName}

	case opType == resources.OpMountDevice:
		details = mountDevice(args[1], args[2], utils.GetJSONArgs(args[3]))
		resp = resources.Response{Status: details["status"], Message: details["msg"]}

	case opType == resources.OpDetach:
		details = detach(args[1], args[2])
		resp = resources.Response{Status: details["status"], Message: details["msg"]}

	case opType == resources.OpWaitForDetach:
		details = waitForDetach(args[1])
		resp = resources.Response{Status: details["status"], Message: details["msg"]}

	case opType == resources.OpUnmountDevice:
		details = unmountDevice(args[1])
		resp = resources.Response{Status: details["status"], Message: details["msg"]}

	case opType == resources.OpMount:
		details = mount(args[1], utils.GetJSONArgs(args[2]))
		resp = resources.Response{Status: details["status"], Message: details["msg"]}

	case opType == resources.OpUnmount:
		details = unmount(args[1])
		resp = resources.Response{Status: details["status"], Message: details["msg"]}
	}
	return resp
}

// Driver entry point
func main() {
	var resp resources.Response
	// Parse out the prefix that they chose to use for the flex volume command
	resources.UpdateDriverPrefix(strings.TrimSuffix(filepath.Base(os.Args[0]), "-volume-flex"))
	utils.SetupLogging()
	var args = os.Args[1:]
	var opType = args[0]
	log.Debugf("The args to main are %s %s \n", opType, args)
	isValid, msg := utils.ValidateArgs(args)
	if !isValid {
		resp = resources.Response{
			Status:  resources.ResultStatusFailed,
			Message: msg,
		}
	} else {
		// Handle the operation
		resp = createRespMsg(opType, args)
	}
	if res, err := json.Marshal(resp); err == nil {
		log.Infof("Returning response %s", res)
		// Output to console to be read by the controller.
		fmt.Println(string(res))
	} else {
		fmt.Println(`{"status": "Failed"}`)
	}
	utils.CloseLogFile()
}
