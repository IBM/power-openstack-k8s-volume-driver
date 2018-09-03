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
	"errors"
	"fmt"
	"strings"

	resources "github.com/IBM/power-openstack-k8s-volume-driver/pkg/resources"
	utils "github.com/IBM/power-openstack-k8s-volume-driver/pkg/utils"

	"github.com/golang/glog"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/volumes"
	"github.com/kubernetes-incubator/external-storage/lib/controller"
	"github.com/kubernetes-incubator/external-storage/lib/util"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type openstackProvisioner struct {
	// The unique name for this provisioner
	ProvisionerName string

	Client kubernetes.Interface
}

// We need to be able to add the multi-attach attribute to the volume creation
type volumeCreateOpts struct {
	Name             string `json:"name,omitempty"`
	Size             int    `json:"size" required:"true"`
	VolumeType       string `json:"volume_type,omitempty"`
	AvailabilityZone string `json:"availability_zone,omitempty"`
	MultiAttach      bool   `json:"multiattach,omitempty"`
}

func (opts volumeCreateOpts) ToVolumeCreateMap() (map[string]interface{}, error) {
	return gophercloud.BuildRequestBody(opts, "volume")
}

// creates and returns a new provisioner
func NewOpenstackProvisioner(client kubernetes.Interface, provisionerName string) (controller.Provisioner, error) {
	provisioner := &openstackProvisioner{
		ProvisionerName: provisionerName,
		Client:          client,
	}
	return provisioner, nil
}

func (p *openstackProvisioner) Provision(options controller.VolumeOptions) (*v1.PersistentVolume, error) {

	opts, fsType, err := p.parseOptions(options)
	if err != nil {
		glog.Errorf("Failed to parse volume options: %s", err)
		return nil, err
	}

	annotations := make(map[string]string)
	annotations[resources.K8sCreatedBy] = resources.ProvisionerNameOnly

	// options.Parameters "test" means we are using a test PVC and want to mock the cinder client
	// if that isn't set, we want to pass
	testParam := ""
	if _, ok := options.Parameters["test"]; ok {
		testParam = "test"
	}
	// if we are testing, we want to pass that information to the PV so we know to use a mock client on delete
	if testParam == "test" {
		annotations["test"] = "test"
	}
	// Creates a new OpenStack Cinder Client and authenticates
	cinderClient, err := utils.CreateCinderClient(testParam)
	if err != nil {
		glog.Errorf("Failed to construct / authenticate OpenStack : %s", err)
		return nil, err
	}

	// creates the volume
	volume, err := volumes.Create(cinderClient, opts).Extract()
	if err != nil {
		glog.Errorf("Failed to provision the volume: %s", err)
		return nil, err
	}

	// If the volume isn't still created yet, we need to wait until it is created
	if volume.Status != "available" {
		// Query the volume and wait for it to actually get fully created
		updVolume, err := utils.GetCinderVolume(cinderClient, volume.ID)
		if err != nil {
			glog.Errorf("Failed to schedule and create the volume: %s", err)
			return nil, err
		}
		if updVolume.Status == "error" {
			err = errors.New("Unknown error creating volume")
			if updVolume.Metadata["schedule Failure description"] != "" {
				err = errors.New(updVolume.Metadata["schedule Failure description"])
			}
			// Clean up the volume we just created since it will be orphaned otherwise
			volumes.Delete(cinderClient, volume.ID)
			glog.Errorf("Failed to schedule and create the volume: %s", err)
			return nil, err
		}
	}

	glog.Infof("Volume %s has been created with the following specs: %s", volume.ID, volume)
	annotations["volumeID"] = volume.ID

	flexVolumeOptions := make(map[string]string)
	flexVolumeOptions["volumeID"] = volume.ID

	// Since the ReadOnly flag in the isn't honored currently for the kubernetes.io/readwrite
	// argument, we will add our own flag to go off of for now until the other one is fixed
	flexVolumeOptions[resources.OsArgsMountRW] = "rw"
	if util.AccessModesContains(options.PVC.Spec.AccessModes, v1.ReadOnlyMany) {
		flexVolumeOptions[resources.OsArgsMountRW] = "ro"
	}

	pv := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:        options.PVName,
			Labels:      map[string]string{},
			Annotations: annotations,
		},
		Spec: v1.PersistentVolumeSpec{
			PersistentVolumeReclaimPolicy: options.PersistentVolumeReclaimPolicy,
			AccessModes:                   options.PVC.Spec.AccessModes,
			Capacity: v1.ResourceList{
				v1.ResourceName(v1.ResourceStorage): resource.MustParse(fmt.Sprintf("%dGi", opts.Size)),
			},
			PersistentVolumeSource: v1.PersistentVolumeSource{
				FlexVolume: &v1.FlexVolumeSource{
					Driver:  resources.FlexPluginVendorDriver,
					Options: flexVolumeOptions,
					// We want the file system mounted as read-only if they assed for read-only-many
					ReadOnly: util.AccessModesContains(options.PVC.Spec.AccessModes, v1.ReadOnlyMany),
					FSType:   fsType,
				},
			},
		},
	}
	return pv, nil
}

// Parses the volume options to populate a struct for the gophercloud create call
func (p *openstackProvisioner) parseOptions(options controller.VolumeOptions) (volumeCreateOpts, string, error) {
	var createOptions volumeCreateOpts
	if options.PVC == nil {
		return createOptions, "", fmt.Errorf("volume options are missing PVC")
	}

	if options.PVName == "" {
		return createOptions, "", fmt.Errorf("volume options are missing PVName")
	}

	capacity, ok := options.PVC.Spec.Resources.Requests[v1.ResourceName(v1.ResourceStorage)]
	if !ok {
		return createOptions, "", fmt.Errorf("volume options are missing storage capactiy in PVC spec")
	}
	sizeGB := int(util.RoundUpSize(capacity.Value(), util.GiB))
	glog.Infof("Volume requested with %dGB", sizeGB)

	fsType := ""
	volumeType := ""
	availabilityZone := ""
	// This is supported in the openstack cinder provisioner, so we may want to include it as well
	for key, value := range options.Parameters {
		switch strings.ToLower(key) {
		case "type":
			volumeType = value
		case "availability":
			availabilityZone = value
		// We want to make sure to let the fsType option flow through to the flex volume driver
		case "fstype":
			fsType = value
		// This means we are testing, go ahead
		case "test":
			continue
		default:
			return createOptions, "", fmt.Errorf("volume options unknown parameter passed in: %s", key)
		}
	}
	if volumeType == "" {
		glog.Info("StorageClass parameter, type, is empty")
	}
	if availabilityZone == "" {
		glog.Info("StorageClass parameter, availability, is empty")
	}
	// Determine if this is ReadWriteMany or ReadOnlyMany so that we specify if we want mult-attach
	multiAttach := util.AccessModesContains(options.PVC.Spec.AccessModes, v1.ReadWriteMany)
	multiAttach = multiAttach || util.AccessModesContains(options.PVC.Spec.AccessModes, v1.ReadOnlyMany)

	return volumeCreateOpts{
		// We want to always name the volume with the ICP prefix for clarity
		Name:             fmt.Sprintf("icp-%s", options.PVName),
		Size:             sizeGB,
		VolumeType:       volumeType,
		AvailabilityZone: availabilityZone,
		MultiAttach:      multiAttach,
	}, fsType, nil
}
