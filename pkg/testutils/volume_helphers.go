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
package testutils

import (
	"strings"

	"github.com/kubernetes-incubator/external-storage/lib/controller"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Creates fake volume options based on based in parameters
func MockVolumeOptions(policy v1.PersistentVolumeReclaimPolicy, name string, pvc *v1.PersistentVolumeClaim, parameters map[string](string)) controller.VolumeOptions {
	options := controller.VolumeOptions{
		PersistentVolumeReclaimPolicy: policy,
		PVName:     name,
		PVC:        pvc,
		Parameters: parameters,
	}
	return options
}

func MockReclaimPolicy() v1.PersistentVolumeReclaimPolicy {
	return v1.PersistentVolumeReclaimDelete
}

// Creates a new Persistent Volume Claim resource we can test on
func MockPVC(testFlags ...string) *v1.PersistentVolumeClaim {
	claim := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test",
			Annotations: map[string]string{"test": "test"},
		},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadOnlyMany, v1.ReadWriteOnce},
			Resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceName(v1.ResourceStorage): resource.MustParse("1Gi"),
				},
			},
		},
	}
	for _, flag := range testFlags {
		switch strings.ToLower(flag) {
		case "capacity":
			claim.Spec.Resources.Requests = nil
		default:
		}
	}
	return claim
}

// Creates a new Persistent Volume resource we can test on
func MockPV() *v1.PersistentVolume {
	pv := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "ibm/test-provisioner",
			Labels: map[string]string{},
			Annotations: map[string]string{
				"kubernetes.io/createdby": "power-openstack-k8s-volume-provisioner",
				"test":     "test",
				"volumeID": "icp-test",
			},
		},
		Spec: v1.PersistentVolumeSpec{
			PersistentVolumeSource: v1.PersistentVolumeSource{
				FlexVolume: &v1.FlexVolumeSource{
					Driver:   "ibm/power-openstack-k8s-volume-flex",
					ReadOnly: true,
					Options: map[string]string{
						"actualReadWrite": "ro",
						"volumeID":        "icp-test",
					},
				},
				Local: nil,
			},
			ClaimRef:                      nil,
			PersistentVolumeReclaimPolicy: "Delete",
			VolumeMode:                    nil,
		},
	}
	return pv
}
