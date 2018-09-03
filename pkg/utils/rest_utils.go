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
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"time"

	resources "github.com/IBM/power-openstack-k8s-volume-driver/pkg/resources"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/hypervisors"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/volumeattach"
	"github.com/gophercloud/gophercloud/pagination"

	netutil "k8s.io/apimachinery/pkg/util/net"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	volumes_v3 "github.com/gophercloud/gophercloud/openstack/blockstorage/v3/volumes"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	ports_v2 "github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
)

var (
	log = Log
	// Fake Token/Server objects to help with any tests run
	FakeToken  = "01234567890123456789012345678901"
	FakeServer *httptest.Server
)

// OpenstackCloudI : Interface that defines openstack cloud API methods
type OpenstackCloudI interface {
	GetAllOSVMs() (*[]resources.OSServer, error)
	GetOSVolumeByID(volumeID string) (*resources.OSVolume, error)
	GetStorageHostRegistration(hostname string) (*resources.StorageRegistration, error)
	AttachVolumeToVM(vmID string, volumeID string, volume *resources.OSVolume) (bool, error)
	DetachVolumeFromVM(vmID string, volumeID string, volume *resources.OSVolume) (bool, error)
	IsVolumeAttached(vmID string, volumeID string) (bool, error)
	GetVolumeByMetadataProperty(volumeMeta map[string]string) (*[]resources.OSVolume, error)
	UpdateVolumeMetadata(volumeID string, volumeMeta map[string]string, isDelete bool) error
	ListHypervisors() (*[]hypervisors.Hypervisor, error)
	GetServerIDFromNodeName(nodeName string) (string, error)
	GetProviderClient() *gophercloud.ProviderClient
}

// OpenstackCloud : Reference to openstack provider
type OpenstackCloud struct {
	Provider *gophercloud.ProviderClient
}

// Fake Endpoint object to help with any tests run
func FakeEndpoint() string {
	return FakeServer.URL + "/"
}

// Fake ServiceClient object to help with any tests run
func FakeServiceClient() *gophercloud.ServiceClient {
	return &gophercloud.ServiceClient{
		Endpoint:       FakeEndpoint(),
		ProviderClient: &gophercloud.ProviderClient{TokenID: FakeToken},
	}
}

// CreateOpenstackClient : Create an OpenStack Client and Authenticate to OpenStack
func CreateOpenstackClient(testParam ...string) (OpenstackCloudI, error) {
	// Load the Environment Variables from the Configuration File
	loadConfigFile()
	// The configuration info is in environment variables with the openstack names
	opts, err := openstack.AuthOptionsFromEnv()
	if err != nil {
		return nil, fmt.Errorf("Error while reading config options %s", err.Error())
	}
	// Construct a new OpenStack Rest Client with the Authentication URL
	providerClient, err := openstack.NewClient(opts.IdentityEndpoint)
	if err != nil {
		return nil, fmt.Errorf("Error while constructing client from openstack %s", err.Error())
	}
	// Update the Rest Client to set the Certificate to use for Validation
	setCertificateOnClient(providerClient)
	// Authenticate to Keystone on the OpenStack controller before using
	err = openstack.Authenticate(providerClient, opts)
	if err != nil {
		return nil, fmt.Errorf("Error while authenticating from openstack %s", err.Error())
	}
	log.Debugf("Authentication with openstack succcessful")
	openStackController := OpenstackCloud{
		Provider: providerClient,
	}
	return &openStackController, nil
}

// CreateCinderClient : Create a Cinder Service Client and Authenticate to OpenStack
func CreateCinderClient(testParam ...string) (*gophercloud.ServiceClient, error) {
	// testParam[0] will be "test" if we are testing, where we want to return a fake client,
	// or "" in which case we want to skip and continue normally
	if testParam[0] != "" {
		return FakeServiceClient(), nil
	}
	openstackClient, err := CreateOpenstackClient()
	if err != nil {
		return nil, err
	}
	// Get a Cinder ServiceClient
	serviceClient, err := openstack.NewBlockStorageV3(
		openstackClient.GetProviderClient(), gophercloud.EndpointOpts{})
	if err != nil {
		return nil, err
	}
	return serviceClient, nil
}

// GetProviderClient : Returns back the embedded ProviderClient in the Interface
func (opnStk *OpenstackCloud) GetProviderClient() *gophercloud.ProviderClient {
	return opnStk.Provider
}

// GetAllOSVMs : Returns list of all VMs from Openstack
func (opnStk *OpenstackCloud) GetAllOSVMs() (*[]resources.OSServer, error) {
	novaClient, err := opnStk.NewComputeV2()
	if err != nil {
		return nil, err
	}
	listOpts := servers.ListOpts{
		TenantID: os.Getenv("OS_PROJECT_NAME"),
	}

	var vmList []resources.OSServer
	allPages, err := servers.List(novaClient, listOpts).AllPages()
	if err != nil {
		return nil, err
	}
	err = servers.ExtractServersInto(allPages, &vmList)
	if err != nil {
		log.Errorf("Could not get VM data. Error is %s", err)
		return nil, err
	}
	log.Debugf("VM details %s", vmList[0].HypervisorHostname)
	return &vmList, nil
}

// GetOSVolumeByID : Function returns volume given volume id
func (opnStk *OpenstackCloud) GetOSVolumeByID(volumeID string) (*resources.OSVolume, error) {
	cinderClient, err := opnStk.NewVolumeV3()
	if err != nil {
		return nil, err
	}
	return GetCinderVolume(cinderClient, volumeID)
}

// GetCinderVolume : Function returns volume given volume id
func GetCinderVolume(cinderClient *gophercloud.ServiceClient, volumeID string) (*resources.OSVolume, error) {
	var volume resources.OSVolume
	// We will retrieve the volume but want to wait until it has finished creating, so we
	// will check the status and if not-ready, then wait and loop and try retrieving again
	for i := 0; i < 100; i++ {
		res := volumes_v3.Get(cinderClient, volumeID)
		err := res.ExtractInto(&volume)
		if err != nil {
			log.Errorf("Could not get volume info. Error is %s", err)
			return nil, err
		}
		// If the volume is still in creating, it isn't ready yet
		if volume.Status != "creating" {
			return &volume, nil
		}
		// Sleep for 2 seconds
		time.Sleep(3 * time.Second)
	}
	return &volume, nil
}

// AttachVolumeToVM : attaches volume to VM
func (opnStk *OpenstackCloud) AttachVolumeToVM(vmID string, volumeID string, volume *resources.OSVolume) (bool, error) {
	// Attach the volume now
	novaClient, err := opnStk.NewComputeV2()
	if err != nil {
		return false, err
	}
	_, err = volumeattach.Create(novaClient, vmID, &volumeattach.CreateOpts{VolumeID: volumeID}).Extract()
	if err != nil {
		log.Errorf("Failed to attach volume %s to VM %s. Error is %s", volumeID, vmID, err)
		return false, err
	}
	return true, nil
}

// DetachVolumeFromVM : detaches volume from VM
func (opnStk *OpenstackCloud) DetachVolumeFromVM(vmID string, volumeID string, volume *resources.OSVolume) (bool, error) {
	var err error
	novaClient, err := opnStk.NewComputeV2()
	if err != nil {
		return false, err
	}
	err = volumeattach.Delete(novaClient, vmID, volume.ID).ExtractErr()
	if err != nil {
		log.Errorf("Failed to remove volume %s from VM %s. Error is %s", volumeID, vmID, err)
		return false, err
	}
	return true, nil
}

// IsVolumeAttached : Determines if a volume is attached to a VM
func (opnStk *OpenstackCloud) IsVolumeAttached(vmID string, volumeID string) (bool, error) {
	novaClient, err := opnStk.NewComputeV2()
	if err != nil {
		return false, err
	}
	attachment, err := volumeattach.Get(novaClient, vmID, volumeID).Extract()
	if err != nil {
		log.Errorf("Error querying if volume %s is attached to VM %s. Error is %s", volumeID, vmID, err)
		return false, err
	}
	if attachment.ServerID == "" {
		log.Warningf("Volume %s is not attached to any VM.", volumeID)
		return false, nil
	} else if vmID != attachment.ServerID {
		log.Warningf("Volume %s is already attached to different VM %s", volumeID, vmID)
		return false, nil
	} else if vmID == attachment.ServerID {
		log.Infof("Volume %s is attached to VM %s", volumeID, vmID)
		return true, nil
	}
	return false, nil
}

// GetVolumeByMetadataProperty : Retrieves Openstack cinder volume by querying its metadata
func (opnStk *OpenstackCloud) GetVolumeByMetadataProperty(volumeMeta map[string]string) (*[]resources.OSVolume, error) {
	var volList []resources.OSVolume
	cinderClient, err := opnStk.NewVolumeV3()
	if err != nil {
		return nil, err
	}
	reqOptList := volumes_v3.ListOpts{
		Metadata:   volumeMeta,
		TenantID:   os.Getenv("OS_PROJECT_NAME"),
		AllTenants: false,
	}
	err = volumes_v3.List(cinderClient, reqOptList).EachPage(
		func(page pagination.Page) (bool, error) {
			var volSubList []resources.OSVolume
			err := volumes_v3.ExtractVolumesInto(page, &volSubList)
			if err != nil {
				log.Errorf("Could not extract volume details. Err is %s", err)
				return false, err
			}
			for _, volume := range volSubList {
				volList = append(volList, volume)
			}
			return true, nil
		})
	if err != nil {
		log.Errorf("Could not get volume details for %s. Err is %s", volumeMeta, err)
		return nil, err
	}
	if len(volList) == 0 {
		err = errors.New(fmt.Sprintf("Can't find volume mapping for %s.", volumeMeta))
		return nil, err
	}
	// If there was more than one volume matching, something must have went wrong, so can't figure out which
	if len(volList) > 1 {
		err = errors.New(fmt.Sprintf("More than one volume mapping for %s.", volumeMeta))
		return nil, err
	}
	log.Debugf("Volume details %s", volList)
	return &volList, nil
}

// UpdateVolumeMetadata : Updates volume metadata
func (opnStk *OpenstackCloud) UpdateVolumeMetadata(volumeID string,
	volumeMeta map[string]string, isDelete bool) error {
	cinderClient, err := opnStk.NewVolumeV3()
	if err != nil {
		return err
	}
	reqOpts := gophercloud.RequestOpts{OkCodes: []int{200}}
	metaKey, result := resources.OsK8sVolumeNameMeta, gophercloud.Result{}
	metaURL := fmt.Sprintf("%s/volumes/%s/metadata", cinderClient.ResourceBaseURL(), volumeID)
	// Depending on if this is a delete or an update we need to call a different Metadata operation
	if !isDelete {
		var reqBody = map[string]map[string]string{}
		reqBody["metadata"] = volumeMeta
		_, err := cinderClient.Post(metaURL, reqBody, &result.Body, &reqOpts)
		if err != nil {
			return err
		}
	} else {
		_, err = cinderClient.Delete(metaURL+"/"+metaKey, &reqOpts)
	}
	log.Debugf("Updated volume metadata details ")
	return nil
}

// ListHypervisors : Returns hypervisor list
func (opnStk *OpenstackCloud) ListHypervisors() (*[]hypervisors.Hypervisor, error) {
	novaClient, err := opnStk.NewComputeV2()
	var hostList []hypervisors.Hypervisor
	if err != nil {
		return nil, err
	}

	hostPager := hypervisors.List(novaClient)
	err = hostPager.EachPage(func(page pagination.Page) (bool, error) {
		hosts, err := hypervisors.ExtractHypervisors(page)
		if err != nil {
			return false, err
		}
		hostList = append(hostList, hosts...)
		return true, nil
	})
	if err != nil {
		log.Errorf("Could not get list of hypervisors. Error is %s", err)
		return nil, err
	}
	return &hostList, nil
}

// GetStorageHostRegistration : Returns storage host registration by name
func (opnStk *OpenstackCloud) GetStorageHostRegistration(hostname string) (*resources.StorageRegistration, error) {
	openstackClient, err := opnStk.NewVolumeV3()
	if err != nil {
		return nil, err
	}
	r := gophercloud.Result{}
	// Get cinder base url
	baseURL := openstackClient.ResourceBaseURL()
	// Make a call to Get os-hosts/hostname
	_, r.Err = openstackClient.Get(baseURL+"os-hosts/"+hostname, &r.Body, nil)
	if r.Err != nil {
		log.Error(r.Err.Error())
		return nil, r.Err
	}
	// Extract the body into our storage registration struct
	log.Debug(fmt.Sprintf("%s", r.Body))
	var storageHost map[string][]map[string]resources.StorageRegistration
	err = r.ExtractInto(&storageHost)
	if err != nil {
		log.Error(err.Error())
	}
	for _, v := range storageHost["host"] {
		if storageReg, ok := v["registration"]; ok {
			return &storageReg, nil
		}
	}
	return nil, nil
}

// GetServerIDFromNodeName : Returns VM ID given its IP
func (opnStk *OpenstackCloud) GetServerIDFromNodeName(nodeName string) (string, error) {
	var portList []ports_v2.Port
	// First lets convert the node name to an IP Address to lookup the VM based on
	nodeAddress := ResolveNodeAddress(nodeName)
	if nodeAddress == "" {
		return "", fmt.Errorf("Unable to determine IP address for node %s", nodeName)
	}
	neutronClient, err := opnStk.NewNetworkV2()
	if err != nil {
		return "", err
	}
	portURL := fmt.Sprintf("%s/ports?fixed_ips=ip_address=%s", neutronClient.ResourceBaseURL(), nodeAddress)
	// Make the rest call to query all of the Ports but filtering on the IP Address given to us
	portPager := pagination.NewPager(neutronClient, portURL, func(r pagination.PageResult) pagination.Page {
		return ports_v2.PortPage{
			LinkedPageBase: pagination.LinkedPageBase{PageResult: r},
		}
	})
	// Loop through each of the pages extracting the port information (which should hopefully be 1 returned)
	err = portPager.EachPage(func(page pagination.Page) (bool, error) {
		portSubList, err := ports_v2.ExtractPorts(page)
		if err != nil {
			return false, fmt.Errorf("Unable to extract ports for the IP. Error is %s", err.Error())
		}
		// Loop through each of the ports on the page adding it to the list that we return
		for _, port := range portSubList {
			portList = append(portList, port)
		}
		return true, nil
	})
	if err != nil {
		return "", fmt.Errorf("Unable to query ports for the given IP. Error is %s", err.Error())
	}
	// We need to also make sure there is a matching port and only one, otherwise we don't know the ID
	if len(portList) == 0 {
		return "", fmt.Errorf("Unable to find matching server for given IP %s", nodeAddress)
	}
	if len(portList) > 1 {
		return "", fmt.Errorf("Found more than one matching port for given IP %s", nodeAddress)
	}
	return portList[0].DeviceID, nil
}

// NewComputeV2 :  Returns nova service client
func (opnStk *OpenstackCloud) NewComputeV2() (*gophercloud.ServiceClient, error) {
	client, err := openstack.NewComputeV2(opnStk.Provider, gophercloud.EndpointOpts{})
	if err != nil {
		log.Errorf("Could not get openstack nova client. Error is %s", err)
		return nil, err
	}
	return client, nil
}

// NewVolumeV3 : Returns cinder service client
func (opnStk *OpenstackCloud) NewVolumeV3() (*gophercloud.ServiceClient, error) {
	client, err := openstack.NewBlockStorageV3(opnStk.Provider, gophercloud.EndpointOpts{})
	if err != nil {
		log.Errorf("Could not get openstack cinder client. Error is %s", err)
		return nil, err
	}
	return client, nil
}

// NewNetworkV2 : Returns Neutron service client
func (opnStk *OpenstackCloud) NewNetworkV2() (*gophercloud.ServiceClient, error) {
	client, err := openstack.NewNetworkV2(opnStk.Provider, gophercloud.EndpointOpts{})
	if err != nil {
		log.Errorf("Could not get openstack neutron client. Error is %s", err)
		return nil, err
	}
	return client, nil
}

func setCertificateOnClient(client *gophercloud.ProviderClient) {
	config := &tls.Config{}
	// If we were given a certificate file, use it, otherwise don't do validation
	if certPath, ok := os.LookupEnv("OS_CACERT"); ok && certPath != "" {
		caPool := x509.NewCertPool()
		caPool.AppendCertsFromPEM(readCertificate(certPath))
		config.RootCAs = caPool
	} else {
		config.InsecureSkipVerify = true
	}
	// Need to update the existing transport to include the additional certificate config
	client.HTTPClient.Transport = netutil.SetOldTransportDefaults(&http.Transport{TLSClientConfig: config})
}

// readCertificate :  Read the openstack server certificate file
func readCertificate(certFile string) []byte {
	certData, err := ioutil.ReadFile(certFile)
	if err != nil {
		Log.Errorf("Could not load Openstack certificate. Error is %s", err)
		return nil
	}
	// Since the golang x509 library doesn't properly handle spaces in the certificate
	// rather than new lines, we will do some trickery to convert the spaces to new
	// lines, but to do so need to temporarily replace real spaces so they don't get hit
	certDataStr := strings.Replace(string(certData), " CERTIFICATE", ".CERTIFICATE", -1)
	certDataStr = strings.Replace(certDataStr, " ", "\n", -1)
	certDataStr = strings.Replace(certDataStr, ".CERTIFICATE", " CERTIFICATE", -1)
	return []byte(certDataStr)
}

func loadConfigFile() {
	// The configuration file is in the same directory as this program
	cmdDir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	confFile := fmt.Sprintf("%s/%s.conf", cmdDir, filepath.Base(os.Args[0]))
	// We can only parse the configuration file if it exists on the system
	if _, err := os.Stat(confFile); !os.IsNotExist(err) {
		afile, _ := os.Open(confFile)
		defer afile.Close()
		scanner := bufio.NewScanner(afile)
		// Loop through each of the config variable lines in the file
		for scanner.Scan() {
			line := scanner.Text()
			index := strings.Index(line, "=")
			// Parse the key/value pair and set as an environment variable
			if index > 0 {
				key, val := strings.TrimSpace(line[0:index]), strings.TrimSpace(line[index+1:])
				// If the variable ends with _ENC it is encoded, so decode it now
				if strings.HasSuffix(key, "_ENC") {
					bval, _ := base64.URLEncoding.DecodeString(val)
					key, val = key[:len(key)-4], strings.TrimSpace(string(bval))
				}
				os.Setenv(key, val)
			}
		}
	}
}
