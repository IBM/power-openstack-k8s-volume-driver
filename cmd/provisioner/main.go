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
package main

import (
	"flag"

	resources "github.com/IBM/power-openstack-k8s-volume-driver/pkg/resources"
	volume "github.com/IBM/power-openstack-k8s-volume-driver/pkg/volume"

	"github.com/golang/glog"
	"github.com/kubernetes-incubator/external-storage/lib/controller"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	prefix = flag.String("prefix", "power-openstack-k8", "The prefix to use for the name of the volume provisioner.")
)

func main() {
	flag.Parse()
	flag.Set("logtostderr", "true")
	resources.UpdateDriverPrefix(*prefix)

	glog.Info("Building kubeconfig for running in cluster")
	config, err := rest.InClusterConfig()
	if err != nil {
		glog.Fatalf("Failed to create config: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		glog.Fatalf("Failed to create client: %v", err)
	}

	// The controller needs to know what the server version is because out-of-tree
	// provisioners aren't officially supported until 1.5
	serverVersion, err := clientset.Discovery().ServerVersion()
	if err != nil {
		glog.Fatalf("Error getting server version: %v", err)
	}

	// Create the provisioner that implements the provisoner interface expected by the controller
	openstackProvisioner, err := volume.NewOpenstackProvisioner(clientset, resources.ProvisionerName)
	if err != nil {
		glog.Fatalf("Error creating the %s provisioner: %v", resources.ProvisionerName, err)
	}

	// Start the provisioner controller, which dynamically provisions the PersistentVolumes
	pc := controller.NewProvisionController(
		clientset,
		resources.ProvisionerName,
		openstackProvisioner,
		serverVersion.GitVersion,
	)
	glog.Infof("New provision controller started for %s", resources.ProvisionerName)
	pc.Run(wait.NeverStop)
}
