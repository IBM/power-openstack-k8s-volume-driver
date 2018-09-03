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

	"k8s.io/client-go/kubernetes/fake"
)

func TestDelete(t *testing.T) {
	testutils.SetupHTTP()
	defer testutils.TearDownHTTP()

	pv := testutils.MockPV()

	testutils.MuxHandleDelete(t, pv.Annotations["volumeID"])

	fakeClientset := fake.NewSimpleClientset()
	testProvisioner, err := NewOpenstackProvisioner(fakeClientset, pName)
	if err != nil {
		t.Errorf("failed to create testProvisioner: %v", err)
	}

	err = testProvisioner.Delete(pv)
	if err != nil {
		t.Errorf("failed to delete pv: %s", err)
	}
}
