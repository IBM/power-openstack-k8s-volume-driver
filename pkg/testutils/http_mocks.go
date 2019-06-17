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
package testutils

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	utils "github.com/IBM/power-openstack-k8s-volume-driver/pkg/utils"
)

var Mux *http.ServeMux

func SetupHTTP() {
	Mux = http.NewServeMux()
	utils.FakeServer = httptest.NewServer(Mux)
}

func TearDownHTTP() {
	utils.FakeServer.Close()
}

// register mux for handling volume create
func MuxHandleCreate(t *testing.T) {
	Mux.HandleFunc("/volumes", func(w http.ResponseWriter, r *http.Request) {
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			fmt.Println("err is : ", err)
		}
		fmt.Println("received: ", b)
		w.WriteHeader(http.StatusAccepted)
		fmt.Fprintf(w, `
{
  "volume": {
    "id": "icp-test",
    "status": "available"
  }
}
    `)
	})
}

// Register mux for handling volume delete of given volumeID
func MuxHandleDelete(t *testing.T, volumeID string) {
	volumePath := fmt.Sprintf("/volumes/%s", volumeID)
	Mux.HandleFunc(volumePath, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})
}

func AssertEquals(t *testing.T, received interface{}, expected interface{}) {
	if received != expected {
		t.Errorf("expected: %s \n receieved: %s", expected, received)
	}
}
