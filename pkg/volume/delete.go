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
package volume

import (
	"fmt"

	utils "github.com/IBM/power-openstack-k8s-volume-driver/pkg/utils"

	"github.com/golang/glog"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/volumes"

	"k8s.io/api/core/v1"
)

func (p *openstackProvisioner) Delete(pv *v1.PersistentVolume) error {

	volumeID := pv.Annotations["volumeID"]
	if volumeID == "" {
		glog.Fatalf("Persistent Volume ID not found")
	}

	testParam := ""
	if _, ok := pv.Annotations["test"]; ok {
		testParam = "test"
	}

	// Creates a new OpenStack Cinder Client and authenticates
	cinderClient, err := utils.CreateCinderClient(testParam)
	if err != nil {
		glog.Errorf("Failed to construct / authenticate OpenStack : %s", err)
		return err
	}

	glog.Infof("Deleting Persistent Volume: %s", volumeID)
	err = volumes.Delete(cinderClient, volumeID).Err
	if err != nil {
		return fmt.Errorf("error deleting volume : %s", err)
	}
	glog.Infof("Persistent Volume %s Deleted", volumeID)
	return nil

}
