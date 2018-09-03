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
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"testing"

	resources "github.com/IBM/power-openstack-k8s-volume-driver/pkg/resources"
	utils "github.com/IBM/power-openstack-k8s-volume-driver/pkg/utils"
)

const getVolumeByNameJSONArgs = `{"kubernetes.io/fsType":"ext4","kubernetes.io/pvOrVolumeName":"vol_1","kubernetes.io/readwrite":"rw","volumeID":"vol_1"}`

func TestInit(t *testing.T) {
	result := driverInit()
	// Test that driver supports the controller enabled attach.
	expectedAttach := true
	if result["attach"] != expectedAttach {
		t.Errorf("Expected attach funcitonality at the driver, but the driver does not support it")
	}
}

func TestGetVolumeByName(t *testing.T) {
	jsonArgs := utils.GetJSONArgs(getVolumeByNameJSONArgs)
	result := getVolumeName(jsonArgs)
	if result == nil || result["volName"] == "05a0a13c-f839-4c67-bd02-28b6fb6fada3" {
		t.Errorf("Expected volume name to be returned as %s", jsonArgs[resources.K8sArgPV])
	}
}

func TestIsAttached(t *testing.T) {
	cloud = &utils.OpenstackCloudMock{}
	nodeName := "1.2.3.4"
	jsonArgs := utils.GetJSONArgs(getVolumeByNameJSONArgs)
	result := isAttached(nodeName, jsonArgs)
	if result["attached"] != "true" {
		t.Errorf("Expecting volume to be attached")
	}
	// Test volume is not attached
	nodeName = "1.2.3.5"
	result = isAttached(nodeName, jsonArgs)
	if result["attached"] != "false" {
		t.Errorf("Expecting volume not to be attached")
	}
}

func TestAttach(t *testing.T) {
	cloud = &utils.OpenstackCloudMock{}
	nodeName := "1.2.3.4"
	jsonArgs := utils.GetJSONArgs(getVolumeByNameJSONArgs)
	result := attach(nodeName, jsonArgs)
	expectedDeviceName := resources.PathPVMVIOS + "wwn_1"
	resultantDeviceName := result["deviceName"]
	if expectedDeviceName != resultantDeviceName {
		t.Errorf("Expecting attached volume device name to be %s. Got %s",
			expectedDeviceName, resultantDeviceName)
	}
}

func TestDetach(t *testing.T) {
	cloud = &utils.OpenstackCloudMock{}
	nodeName := "1.2.3.4"
	result := detach("vol_1", nodeName)
	if result["status"] != resources.ResultStatusSuccess {
		t.Errorf("Expected detach to be successful, but got %s", result["msg"])
	}
}

func TestMountDevice(t *testing.T) {
	utils.ExecCommand = fakeExecCommand
	cmdExitStatus = 0
	cmdsExecuted = []string{}
	devicePath := "dev/sdd"
	mountPath := "/kubelet/mount/vol_1"
	result := mountDevice(mountPath, devicePath, utils.GetJSONArgs(getVolumeByNameJSONArgs))
	if result["status"] != resources.ResultStatusSuccess {
		t.Errorf("Expected mountdevice to be successful, but got %s", result["msg"])
	}
	expectedCmdsRun := []string{resources.CMDMkFS + "ext4", devicePath, "-F",
		resources.CMDMkDir, "-p", mountPath,
		resources.CMDMount, devicePath, mountPath,
	}
	for i, c := range expectedCmdsRun {
		if c != cmdsExecuted[i] {
			t.Errorf("Expected %s to run", cmdsExecuted[i])
		}
	}

	// Test negative path
	cmdExitStatus = -1
	cmdsExecuted = []string{}
	result = mountDevice(mountPath, devicePath, utils.GetJSONArgs(getVolumeByNameJSONArgs))
	if result["status"] != resources.ResultStatusFailed {
		t.Errorf("Expected mountdevice to fail, but got %s", result["msg"])
	}
}

func TestMount(t *testing.T) {
	utils.ExecCommand = fakeExecCommand
	cmdExitStatus = 0
	cmdsExecuted = []string{}
	mountDir := "/kubelet/mount"
	volumeMountDir := resources.GlobalMountsDir + "vol_1"
	result := mount(mountDir, utils.GetJSONArgs(getVolumeByNameJSONArgs))
	if result["status"] != resources.ResultStatusSuccess {
		t.Errorf("Expected mountdevice to be successful, but got %s", result["msg"])
	}
	expectedCmdsRun := []string{resources.CMDMkDir, "-p", mountDir,
		resources.CMDMount, "--bind", volumeMountDir, mountDir}
	for i, c := range expectedCmdsRun {
		if c != cmdsExecuted[i] {
			t.Errorf("Expected %s to run", cmdsExecuted[i])
		}
	}

	// Test negative path
	cmdsExecuted = []string{}
	cmdExitStatus = -1
	result = mount(mountDir, utils.GetJSONArgs(getVolumeByNameJSONArgs))
	if result["status"] != resources.ResultStatusFailed {
		t.Errorf("Expected mountdevice to fail, but got %s", result["msg"])
	}
}

func TestUnmountDevice(t *testing.T) {
	utils.ExecCommand = fakeExecCommand
	cmdExitStatus = 0
	mountPath := "/dev/sdd"
	cmdsExecuted = []string{}
	result := unmountDevice(mountPath)
	if result["status"] != resources.ResultStatusSuccess {
		t.Errorf("Expected mountdevice to be successful, but got %s", result["msg"])
	}
	expectedCmdsRun := []string{resources.CMDUnmount, mountPath}
	for i, c := range expectedCmdsRun {
		if c != cmdsExecuted[i] {
			t.Errorf("Expected %s to run", cmdsExecuted[i])
		}
	}

	// Test negative path
	cmdsExecuted = []string{}
	cmdExitStatus = -1
	result = unmountDevice(mountPath)
	if result["status"] != resources.ResultStatusFailed {
		t.Errorf("Expected mountdevice to fail, but got %s", result["msg"])
	}
}

func TestUnmount(t *testing.T) {
	utils.ExecCommand = fakeExecCommand
	cmdExitStatus = 0
	cmdsExecuted = []string{}
	mountDir := "/kubelet/mount"
	result := unmount(mountDir)
	if result["status"] != resources.ResultStatusSuccess {
		t.Errorf("Expected mountdevice to be successful, but got %s", result["msg"])
	}
	t.Logf("The commands executed are %s", cmdsExecuted)
	expectedCmdsRun := []string{resources.CMDUnmount, mountDir}
	for i, c := range expectedCmdsRun {
		if c != cmdsExecuted[i] {
			t.Errorf("Expected %s to run", cmdsExecuted[i])
		}
	}

	// Test negative path
	cmdsExecuted = []string{}
	cmdExitStatus = -1
	result = unmount(mountDir)
	if result["status"] != resources.ResultStatusFailed {
		t.Errorf("Expected mountdevice to fail, but got %s", result["msg"])
	}
}

/*********************** OS commands mocking *****************/

var cmdRunResult = ""
var cmdExitStatus = 0
var cmdsExecuted = []string{}

func fakeExecCommand(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestHelperProcess", "--", command}
	cs = append(cs, args...)
	// Save the command being executed
	cmdsExecuted = append(cmdsExecuted, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1", "GO_CMD_RESULT_STATUS=" + strconv.Itoa(cmdExitStatus)}
	return cmd
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	// some code here to check arguments perhaps?
	fmt.Fprintf(os.Stdout, cmdRunResult)
	exitStatus, _ := strconv.Atoi(os.Getenv("GO_CMD_RESULT_STATUS"))
	os.Exit(exitStatus)
}
