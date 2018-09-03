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
	"os"
	"strings"
	"testing"

	resources "github.com/IBM/power-openstack-k8s-volume-driver/pkg/resources"
)

func setEnvVar() {
	os.Setenv(resources.OSUser, "root")
	os.Setenv(resources.OSPassword, "passw0rd")
	os.Setenv(resources.OSCACert, "openstack.crt")
	os.Setenv(resources.OSAuthURL, "https://auth.url")
}

/*********** Test util code **********/

func TestValidateArgsForInit(t *testing.T) {
	validArgs := []string{"init"}
	testValidateArgs(t, &validArgs, nil)
}

func TestValidateArgsForGetVolName(t *testing.T) {
	validArgs := []string{"getvolumename", `{"kubernetes.io/fsType":"ext4","kubernetes.io/pvOrVolumeName":"myvol", "kubernetes.io/readwrite":"rw","volumeID":"vol_1"}`, "1.2.3.4"}
	testValidateArgs(t, &validArgs, nil)
}

func TestValidateArgsForAttach(t *testing.T) {
	validArgs := []string{"attach", `{"kubernetes.io/fsType":"ext4","kubernetes.io/pvOrVolumeName":"myvol", "kubernetes.io/readwrite":"rw","volumeID":"vol_1"}`, "1.2.3.4"}
	invalidArgs := []string{"attach"}
	testValidateArgs(t, &validArgs, &invalidArgs)
}

func TestValidateArgsForISAttached(t *testing.T) {
	validArgs := []string{"isattached", `{"kubernetes.io/fsType":"ext4","kubernetes.io/pvOrVolumeName":"myvol", "kubernetes.io/readwrite":"rw","volumeID":"vol_1"}`, "1.2.3.4"}
	invalidArgs := []string{"isattached"}
	testValidateArgs(t, &validArgs, &invalidArgs)
}

func TestValidateArgsForWaitForAttach(t *testing.T) {
	validArgs := []string{"waitforattach", `{"kubernetes.io/fsType":"ext4","kubernetes.io/pvOrVolumeName":"myvol", "kubernetes.io/readwrite":"rw","volumeID":"vol_1"}`}
	//invalidArgs := []string{"waitforattach"}
	testValidateArgs(t, &validArgs, nil)
}

func TestValidateArgsForWaitForMountDevice(t *testing.T) {
	validArgs := []string{"mountdevice",
		"/var/lib/kubelet/plugins/kubernetes.io/flexvolume/pvc/cinder_plugin/mounts/nginx-vol",
		"/dev/sdd",
		`{"kubernetes.io/fsType":"ext4","kubernetes.io/pvOrVolumeName":"myvol", "kubernetes.io/readwrite":"rw","volumeID":"vol_1"}`}
	invalidArgs := []string{"mountdevice"}
	testValidateArgs(t, &validArgs, &invalidArgs)
}

func TestValidateArgsForWaitForMount(t *testing.T) {
	validArgs := []string{"mount",
		"/var/lib/kubelet/pods/4f9b2dd9-08d2-11e8-a53f-fa969e44c120/volumes/pvc~cinder_plugin/nginx-vol",
		`{"kubernetes.io/fsType":"ext4","kubernetes.io/pvOrVolumeName":"myvol", "kubernetes.io/readwrite":"rw","volumeID":"vol_1"}`}
	invalidArgs := []string{"mount"}
	testValidateArgs(t, &validArgs, &invalidArgs)
}

func TestValidateArgsForWaitForUnmount(t *testing.T) {
	validArgs := []string{"unmount",
		"/var/lib/kubelet/pods/4f9b2dd9-08d2-11e8-a53f-fa969e44c120/volumes/pvc~cinder_plugin/nginx-vol",
	}
	invalidArgs := []string{"unmount"}
	testValidateArgs(t, &validArgs, &invalidArgs)
}

func TestValidateArgsForWaitForDetach(t *testing.T) {
	validArgs := []string{"detach", "nginx-vol", "1.2.3.4"}
	invalidArgs := []string{"detach"}
	testValidateArgs(t, &validArgs, &invalidArgs)
}

func testValidateArgs(t *testing.T, validArgs *[]string, invalidArgs *[]string) {
	if invalidArgs != nil {
		isValid, _ := ValidateArgs(*invalidArgs)
		if isValid {
			t.Errorf("Expecting args not to be valid.")
		}
	}

	isValid, _ := ValidateArgs(*validArgs)
	if !isValid {
		t.Errorf("Expecting args to be valid.")
	}
}

func TestGetVMID(t *testing.T) {
	cloud := &OpenstackCloudMock{}
	vmID, err := GetVMID(cloud, "1.2.3.4")
	if err != nil {
		t.Errorf("Error while finding id for VM. Error is %s", err)
	}
	if vmID != "vm_1" {
		t.Error("Found incorrect VM")
	}
}

func TestGetVMIDThruNova(t *testing.T) {
	cloud := &OpenstackCloudMock{}
	vmID, err := GetVMIDThruNova(cloud, "1.2.3.4")
	if err != nil {
		t.Errorf("Error while finding id for VM. Error is %s", err)
	}
	if vmID != "vm_1" {
		t.Error("Found incorrect VM")
	}
}

func TestCreateHostnameToDetailsMap(t *testing.T) {
	cloud := &OpenstackCloudMock{}
	hyps, err := CreateHostnameToDetailsMap(cloud)
	if err != nil {
		t.Errorf("Error while creating the hypervisor map. Error is %s", err)
	}
	if _, ok := hyps["host_1"]; !ok {
		t.Errorf("Expected host_1 to be part of hypervisor map but couldnt find it.")
	}
	for _, hyp := range hyps {
		t.Logf("%s : %s", (*hyp).HypervisorHostname, (*hyp).HypervisorType)
	}
}

func TestGetDirectoryName(t *testing.T) {
	cloud := &OpenstackCloudMock{}
	dirName, err := GetVolumeDirectoryName(cloud, "1.2.3.4", "vol_1", nil)
	if err != nil {
		t.Errorf("Encounted error while getting volume directory name. Error is %s", err)
	}
	expectedDirName := resources.PathPVMVIOS + "wwn_1"
	if dirName != expectedDirName {
		t.Errorf("Expected directory name to be %s, but got %s", expectedDirName, dirName)
	}

	dirName, err = GetVolumeDirectoryName(cloud, "1.2.3.5", "vol_2", nil)
	if err != nil {
		t.Errorf("Encounted error while getting volume directory name. Error is %s", err)
	}
	expectedDirName = resources.AttachedVolumeDir + fmt.Sprintf(resources.DirNamePrefixKVM+"%s", "9ddd4949-5117-4ad8-82b2-8ab37b690078"[:20])
	if dirName != expectedDirName {
		t.Errorf("Expected directory name to be %s, but got %s", expectedDirName, dirName)
	}

	dirName, err = GetVolumeDirectoryName(cloud, "1.2.3.6", "vol_3", nil)
	if err != nil {
		t.Errorf("Encounted error while getting volume directory name. Error is %s", err)
	}
	expectedDirName = resources.AttachedVolumeDir + resources.DirNamePrefixPVMXIV + "wwn_3"
	if dirName != expectedDirName {
		t.Errorf("Expected directory name to be %s, but got %s", expectedDirName, dirName)
	}

	dirName, err = GetVolumeDirectoryName(cloud, "1.2.3.7", "vol_4", nil)
	if err != nil {
		t.Errorf("Encounted error while getting volume directory name. Error is %s", err)
	}
	expectedDirName = resources.AttachedVolumeDir + fmt.Sprintf(resources.DirNamePrefixPVMLIO+"%s", strings.Replace("9ddd4949-5117-4ad8-82b2-8ab37b690080", "-", "", -1)[:25])
	if dirName != expectedDirName {
		t.Errorf("Expected directory name to be %s, but got %s", expectedDirName, dirName)
	}

	dirName, err = GetVolumeDirectoryName(cloud, "1.2.3.8", "vol_5", nil)
	if err != nil {
		t.Errorf("Encounted error while getting volume directory name. Error is %s", err)
	}
	expectedDirName = resources.PathPVMVIOS + "wwn_5"
	if dirName != expectedDirName {
		t.Errorf("Expected directory name to be %s, but got %s", expectedDirName, dirName)
	}
}
