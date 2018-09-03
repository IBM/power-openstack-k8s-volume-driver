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
package volume

import (
	"testing"

	"github.com/IBM/power-openstack-k8s-volume-driver/pkg/testutils"

	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/fake"
)

const (
	pName = "ibm/test-provisioner"
)

func TestParseOptions(t *testing.T) {
	tests := []struct {
		name       string
		policy     v1.PersistentVolumeReclaimPolicy
		pvc        *v1.PersistentVolumeClaim
		pvname     string
		parameters map[string]string
		expected   string
	}{
		{
			name:       "no PVC in volume options",
			policy:     testutils.MockReclaimPolicy(),
			pvc:        nil,
			pvname:     pName,
			parameters: nil,
			expected:   "volume options are missing PVC",
		},
		{
			name:       "no PVName in volume options",
			policy:     testutils.MockReclaimPolicy(),
			pvc:        testutils.MockPVC(),
			pvname:     "",
			parameters: nil,
			expected:   "volume options are missing PVName",
		},
		{
			name:       "no capacity listed",
			policy:     testutils.MockReclaimPolicy(),
			pvc:        testutils.MockPVC("capacity"),
			pvname:     pName,
			parameters: nil,
			expected:   "volume options are missing storage capactiy in PVC spec",
		},
		{
			name:       "unknown parameter",
			policy:     testutils.MockReclaimPolicy(),
			pvc:        testutils.MockPVC(),
			pvname:     pName,
			parameters: map[string]string{"unknown": "test"},
			expected:   "volume options unknown parameter passed in: unknown",
		},
	}
	fakeClientset := fake.NewSimpleClientset()
	testProvisioner, err := NewOpenstackProvisioner(fakeClientset, pName)
	if err != nil {
		t.Errorf("failed to create testProvisioner: %v", err)
	}
	for _, test := range tests {
		// create a fake volume options struct from the above tests and compare against expected errors
		volumeOptions := testutils.MockVolumeOptions(test.policy, test.pvname, test.pvc, test.parameters)
		_, err := testProvisioner.Provision(volumeOptions)
		if err.Error() != test.expected {
			t.Errorf("expected: %s \n received: %s", test.expected, err)
		}
	}
}

func TestProvision(t *testing.T) {
	testutils.SetupHTTP()
	defer testutils.TearDownHTTP()

	testutils.MuxHandleCreate(t)

	fakeClientset := fake.NewSimpleClientset()
	testProvisioner, err := NewOpenstackProvisioner(fakeClientset, pName)
	if err != nil {
		t.Errorf("failed to create testProvisioner: %v", err)
	}

	volumeOptions := testutils.MockVolumeOptions(testutils.MockReclaimPolicy(), pName, testutils.MockPVC(), map[string]string{"test": "test"})
	pv, err := testProvisioner.Provision(volumeOptions)
	if err != nil {
		t.Errorf("failed to provision volume: %s", err)
	}

	testutils.AssertEquals(t, pv.ObjectMeta.Name, pName)
	testutils.AssertEquals(t, pv.ObjectMeta.Annotations["volumeID"], "icp-test")
	testutils.AssertEquals(t, pv.ObjectMeta.Annotations["kubernetes.io/createdby"], "power-openstack-k8s-volume-provisioner")

	testutils.AssertEquals(t, pv.Spec.PersistentVolumeReclaimPolicy, v1.PersistentVolumeReclaimDelete)
	storage := pv.Spec.Capacity["storage"]
	intCapacity, _ := storage.AsInt64()
	testutils.AssertEquals(t, intCapacity, int64(1073741824))

	testutils.AssertEquals(t, pv.Spec.PersistentVolumeSource.FlexVolume.Driver, "ibm/power-openstack-k8s-volume-flex")
}
