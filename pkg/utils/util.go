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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"

	resources "github.com/IBM/power-openstack-k8s-volume-driver/pkg/resources"

	logging "github.com/op/go-logging"
)

// Pointer to log file
var logFile *os.File

// Log : Global logger
var Log = logging.MustGetLogger(resources.FlexPluginVendor + "-" + resources.FlexPluginDriver)

// ExecCommand : Holds exec.Command object
var ExecCommand = exec.Command

/********************** Utility methods *****************************/

// GetJSONArgs : parses and returns json args
func GetJSONArgs(args string) map[string]string {
	var jsonArgs map[string]string
	json.Unmarshal([]byte(args), &jsonArgs)
	return jsonArgs
}

// ValidateArgs : Validate required arguments for operation
func ValidateArgs(args []string) (bool, string) {
	opArgsMap := createOpArgsMap()
	operationType := args[0]
	opReqsParams := opArgsMap[operationType]["required"]
	if len(args)-1 < len(opReqsParams) {
		msg := fmt.Sprintf("The required number of parameters for %s is %d. "+
			"However, the program received %d paramsters.", operationType, len(opReqsParams), len(args)-1)
		Log.Error(msg)
		return false, msg
	}
	return true, ""
}

// Create map of required parameters by operation type
func createOpArgsMap() map[string]map[string][]string {
	opInitArgs := map[string][]string{
		"required": {},
		"optional": {},
	}
	opGetVolumeNameArgs := map[string][]string{
		"required": {},
		"optional": {resources.DriverJSONArgs},
	}
	opIsAttachedArgs := map[string][]string{
		"required": {resources.NodeName},
		"optional": {resources.DriverJSONArgs},
	}
	opAttachArgs := map[string][]string{
		"required": {resources.NodeName},
		"optional": {resources.DriverJSONArgs},
	}
	opWaitForAttachArgs := map[string][]string{
		"required": {resources.DevicePath},
		"optional": {resources.DriverJSONArgs},
	}
	opMountDeviceArgs := map[string][]string{
		"required": {resources.MountPath, resources.DevicePath},
		"optional": {resources.DriverJSONArgs},
	}
	opMountArgs := map[string][]string{
		"required": {resources.MountDir},
		"optional": {resources.DriverJSONArgs},
	}
	opDetachArgs := map[string][]string{
		"required": {resources.DevicePath, resources.NodeName},
		"optional": {},
	}
	opWaitForDetachArgs := map[string][]string{
		"required": {resources.DevicePath},
		"optional": {},
	}
	opUnmountDeviceArgs := map[string][]string{
		"required": {resources.MountPath},
		"optional": {},
	}
	opUnmountArgs := map[string][]string{
		"required": {resources.MountDir},
		"optional": {},
	}
	operationArgsMap := map[string]map[string][]string{
		resources.OpInit:          opInitArgs,
		resources.OpGetVolumeName: opGetVolumeNameArgs,
		resources.OpIsAttached:    opIsAttachedArgs,
		resources.OpAttach:        opAttachArgs,
		resources.OpWaitForAttach: opWaitForAttachArgs,
		resources.OpMountDevice:   opMountDeviceArgs,
		resources.OpMount:         opMountArgs,
		resources.OpDetach:        opDetachArgs,
		resources.OpWaitForDetach: opWaitForDetachArgs,
		resources.OpUnmountDevice: opUnmountDeviceArgs,
		resources.OpUnmount:       opUnmountArgs,
	}
	return operationArgsMap
}

// ErrorStruct : creates error message structure
func ErrorStruct(msg string) map[string]string {
	return map[string]string{
		"status": resources.ResultStatusFailed,
		"msg":    msg,
	}
}

// GetOSVolumeByID :  Returns Openstack Volume, given its id.
func GetOSVolumeByID(cloud OpenstackCloudI, volumeID string) (*resources.OSVolume, error) {
	return cloud.GetOSVolumeByID(volumeID)
}

// IsVolumeAttached : Check if volume is attached to the VM
func IsVolumeAttached(cloud OpenstackCloudI, vmID string, volumeID string) (bool, error) {
	return cloud.IsVolumeAttached(vmID, volumeID)
}

// DetachVolumeFromVM : detach volume on openstack
func DetachVolumeFromVM(cloud OpenstackCloudI, vmID string,
	volumeID string, volume resources.OSVolume) (bool, error) {
	return cloud.DetachVolumeFromVM(vmID, volumeID, &volume)
}

// AttachVolumeToVM : Attach volume to VM
func AttachVolumeToVM(cloud OpenstackCloudI, vmID string,
	volumeID string, volume *resources.OSVolume) (bool, error) {
	return cloud.AttachVolumeToVM(vmID, volumeID, volume)
}

// UpdateVolumeMetadata : Update volume's metadata at Openstack
func UpdateVolumeMetadata(cloud OpenstackCloudI, volumeID string,
	volumeMeta map[string]string, isDelete bool) error {
	return cloud.UpdateVolumeMetadata(volumeID, volumeMeta, isDelete)
}

// GetVolumeByMetadataProperty : Retrieve volume by querying its metadata
func GetVolumeByMetadataProperty(cloud OpenstackCloudI,
	volumeMeta map[string]string) (*[]resources.OSVolume, error) {
	return cloud.GetVolumeByMetadataProperty(volumeMeta)
}

// GetVMID : Get ID of VM given its IP using Neutron API
func GetVMID(cloud OpenstackCloudI, vmIP string) (string, error) {
	return cloud.GetServerIDFromNodeName(vmIP)
}

// GetVMIDThruNova : Get ID of VM given its IP using Nova API
func GetVMIDThruNova(cloud OpenstackCloudI, vmIP string) (string, error) {
	vmIDIPmap, err := CreateVMIDToIPMap(cloud)
	if err != nil {
		return "", err
	}
	for serverID, vmIPs := range vmIDIPmap {
		for _, ip := range vmIPs {
			if ip == vmIP {
				return serverID, nil
			}
		}
	}
	// If we are here, then we could not find this VM on openstack
	return "", nil
}

// CreateVMIDToIPMap : Function creates map of VM ID to list of its assigned IPs
func CreateVMIDToIPMap(cloud OpenstackCloudI) (map[string][]string, error) {
	vms, err := CreateVMIPToVMDetailsMap(cloud)
	if vms == nil {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	log.Debug(fmt.Sprintf("%v", vms))
	vmIDToIPs := make(map[string][]string)
	for ipAddr, vm := range vms {
		vmIDToIPs[vm.ID] = []string{ipAddr}
	}
	Log.Debug(fmt.Sprintf("%s", vmIDToIPs))
	return vmIDToIPs, nil
}

// CreateVMIPToVMDetailsMap : Creates map of VM IP to VM details
func CreateVMIPToVMDetailsMap(cloud OpenstackCloudI) (map[string](resources.OSServer), error) {
	vms, err := cloud.GetAllOSVMs()
	if vms == nil {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	vmIPToDetails := make(map[string]resources.OSServer)
	for _, vm := range *vms {
		for _, vmNets := range vm.Addresses {
			// addresses": {"network_2230": [{"OS-EXT-IPS-MAC:mac_addr"
			// : "fa:82:c4:ae:cd:20", "version": 4, "addr": "9.47.70.67",
			// "OS-EXT-IPS:type": "fixed"}]},
			if nets, ok := vmNets.([]interface{}); ok {
				log.Debug(fmt.Sprintf("%s", nets))
				for _, vmNet := range nets {
					vmNetEntry := vmNet.(map[string]interface{})
					ipAddr := vmNetEntry["addr"].(string)
					vmIPToDetails[ipAddr] = vm
				}
			}
		}
	}
	return vmIPToDetails, nil
}

// CreateHostnameToDetailsMap : Creates map of hypervisor hostname to host details map
func CreateHostnameToDetailsMap(cloud OpenstackCloudI) (map[string](*resources.Hypervisor), error) {
	hosts, err := cloud.GetHypervisors()
	if err != nil {
		return nil, err
	}
	hostMap := make(map[string]*resources.Hypervisor)
	for _, host := range *hosts {
		// Copy the structure into new one
		hyp := resources.Hypervisor(host)
		hostMap[host.HypervisorHostname] = &hyp
	}
	log.Debug(fmt.Sprintf("%v", hostMap))
	return hostMap, nil
}

// ResolveNodeAddress : Converts the Node Name to an IP Address if not already
func ResolveNodeAddress(nodeName string) string {
	// If the nodeName is already an IP Address then we can just return it
	if net.ParseIP(nodeName) != nil {
		return nodeName
	}
	// Since this isn't an IP Address try to resolve the IP and return the first valid one
	ips, err := net.LookupIP(nodeName)
	// If there was an error or no IPs were found, then we can't resolve it
	if err != nil || ips == nil {
		return ""
	}
	// Loop through each of the IPs returned, returning the first IPv4 one
	for _, ip := range ips {
		if ip.To4() != nil {
			return ip.String()
		}
	}
	return ""
}

// RunCommand : Run shell command
func RunCommand(cmdStr string, cmdArgs []string) (string, string, error) {
	Log.Debugf("Running command %s %s", cmdStr, cmdArgs)
	cmd := ExecCommand(cmdStr, cmdArgs...)
	var cmdOutput, cmdError bytes.Buffer
	cmd.Stdout = &cmdOutput
	cmd.Stderr = &cmdError
	err := cmd.Run()
	Log.Debugf("Command output %s %s", cmdOutput.String(), cmdError.String())
	return cmdOutput.String(), cmdError.String(), err
}

// RunPipedCommands : Run cmd1 | cmd2 on OS
func RunPipedCommands(cmd1 string, cmd1Args []string, cmd2 string, cmd2Args []string) (string, string) {
	Log.Debugf("Running command %s %s | %s %s", cmd1, cmd1Args, cmd2, cmd2Args)
	c1 := ExecCommand(cmd1, cmd1Args...)
	c2 := ExecCommand(cmd2, cmd2Args...)
	rPipe, wPipe := io.Pipe()
	c1.Stdout = wPipe
	c2.Stdin = rPipe

	var cmdOutput, cmdError bytes.Buffer
	c2.Stdout = &cmdOutput
	c2.Stderr = &cmdError

	c1.Start()
	c2.Start()
	c1.Wait()
	wPipe.Close()
	c2.Wait()
	// io.Copy(os.Stdout, &cmdOutput)
	// io.Copy(os.Stderr, &cmdError)

	return cmdOutput.String(), cmdError.String()
}

// SetupLogging : Sets up logging for the driver
func SetupLogging() {

	logName := fmt.Sprintf("/var/log/%s.log", filepath.Base(os.Args[0]))
	logFile, err := os.OpenFile(logName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		// Dont really expect this to happen.
	}
	// Defer the close.
	// defer logFile.Close()
	// Comment out the below for unit test.
	//logBackend := logging.NewLogBackend(os.Stdout, "", 0)
	logBackend := logging.NewLogBackend(logFile, "", 0)
	var format = logging.MustStringFormatter(
		`%{time} %{shortfunc} : %{level:.5s} : %{message}`,
	)
	logBackendFormatter := logging.NewBackendFormatter(logBackend, format)
	logLevel := logging.AddModuleLevel(logBackend)
	logLevel.SetLevel(logging.ERROR, "")
	logging.SetBackend(logLevel, logBackendFormatter)
}

// CloseLogFile : Close our plugin log file
func CloseLogFile() {
	logFile.Close()
}
