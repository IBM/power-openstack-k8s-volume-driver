# Copyright IBM Corp. 2018, 2019.
#
#  Licensed under the Apache License, Version 2.0 (the "License");
#  you may not use this file except in compliance with the License.
#  You may obtain a copy of the License at
#      http://www.apache.org/licenses/LICENSE-2.0
#
#  Unless required by applicable law or agreed to in writing, software
#  distributed under the License is distributed on an "AS IS" BASIS,
#  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#  See the License for the specific language governing permissions and
#  limitations under the License.

IMAGE_REPO = ibmcom
DRIVER_VERSION := 1.0.1
OS_ARCH = $(shell uname -p)
DRIVER_PREFIX := power-openstack-k8s
IMAGE_FLEXPLUGIN = $(DRIVER_PREFIX)-volume-flex
IMAGE_PROVISONER = $(DRIVER_PREFIX)-volume-provisioner
IMAGE_TARFILE = $(DRIVER_PREFIX)-volume-driver-$(OS_ARCH)-$(DRIVER_VERSION).tar

# Since the image architecture is amd64 rather than x86_64, we want to change that
ifeq ($(OS_ARCH), x86_64)
$(eval IMAGE_ARCH = amd64)
else
$(eval IMAGE_ARCH = $(OS_ARCH))
endif

# The default target is to do the build and then run the tests off that build
all: build test
.PHONY: all

build:
	mkdir -p output/images output/flexplugin output/provisioner
	# Copy some of the source files to the output directory where we will do the building
	cp -f docker/Dockerfile.flexplugin output/flexplugin/Dockerfile
	cp -f docker/Dockerfile.provisioner output/provisioner/Dockerfile
	cp -f cmd/flexplugin/setup-power-openstack-k8s-volume-flex.sh output/flexplugin/
	chmod 755 output/flexplugin/setup-power-openstack-k8s-volume-flex.sh
	# Build the golang code for both the flex driver and provisioner and put in the output directory
	CGO_ENABLED=0 GOOS=linux go build -o ./output/flexplugin/power-openstack-k8s-volume-flex ./cmd/flexplugin
	CGO_ENABLED=0 GOOS=linux go build -o ./output/provisioner/power-openstack-k8s-volume-provisioner ./cmd/provisioner
	strip output/flexplugin/power-openstack-k8s-volume-flex output/provisioner/power-openstack-k8s-volume-provisioner
	# Build the docker images for both the flex volume driver and the volume provisioner 
	cd output/flexplugin; docker build -t $(IMAGE_REPO)/$(IMAGE_FLEXPLUGIN)-$(IMAGE_ARCH):$(DRIVER_VERSION) .
	cd output/provisioner; docker build -t $(IMAGE_REPO)/$(IMAGE_PROVISONER)-$(IMAGE_ARCH):$(DRIVER_VERSION) .
	# Tag the docker images without the architecture so that we can save off the image as the equivalent of an multi-arch image
	docker tag $(IMAGE_REPO)/$(IMAGE_FLEXPLUGIN)-$(IMAGE_ARCH):$(DRIVER_VERSION) $(IMAGE_REPO)/$(IMAGE_FLEXPLUGIN):$(DRIVER_VERSION)
	docker tag $(IMAGE_REPO)/$(IMAGE_PROVISONER)-$(IMAGE_ARCH):$(DRIVER_VERSION) $(IMAGE_REPO)/$(IMAGE_PROVISONER):$(DRIVER_VERSION)
	# We need to save off the docker images so that we can transfer them for consumability
	docker image save -o output/images/$(IMAGE_TARFILE) $(IMAGE_REPO)/$(IMAGE_FLEXPLUGIN):$(DRIVER_VERSION) $(IMAGE_REPO)/$(IMAGE_PROVISONER):$(DRIVER_VERSION)
	gzip output/images/$(IMAGE_TARFILE)
	chmod 644 output/images/$(IMAGE_TARFILE).gz
.PHONY: build

test:
	# Run the golang test command to execute all of the nested tests
	CGO_ENABLED=0 GOOS=linux go test ./...
.PHONY: test

clean:
	# Clean up the output director and the docker images if they already exist
	-docker image rm $(IMAGE_REPO)/$(IMAGE_FLEXPLUGIN):$(DRIVER_VERSION)
	-docker image rm $(IMAGE_REPO)/$(IMAGE_PROVISONER):$(DRIVER_VERSION)
	-docker image rm $(IMAGE_REPO)/$(IMAGE_FLEXPLUGIN)-$(IMAGE_ARCH):$(DRIVER_VERSION)
	-docker image rm $(IMAGE_REPO)/$(IMAGE_PROVISONER)-$(IMAGE_ARCH):$(DRIVER_VERSION)
	rm -rf output
.PHONY: clean

docker-login:
ifndef $(and DOCKER_USERNAME, DOCKER_PASSWORD)
	$(error DOCKER_USERNAME and DOCKER_PASSWORD must be defined, required for goal (docker-login))
endif
	@docker login -u $(DOCKER_USERNAME) -p $(DOCKER_PASSWORD) $(DOCKER_SERVER)
.PHONY: docker-login

docker-push-images:
	# We want to push both the flex and provisioner images to the docker registry for the architecture that we built on
	docker push $(IMAGE_REPO)/$(IMAGE_FLEXPLUGIN)-$(IMAGE_ARCH):$(DRIVER_VERSION)
	docker push $(IMAGE_REPO)/$(IMAGE_PROVISONER)-$(IMAGE_ARCH):$(DRIVER_VERSION)
.PHONY: docker-push-images

docker-manifest-tool:
	# We will use the manifest tool to be able to push the manifest list to the docker registry for the multi-arch support
	sudo curl -sSL -o /usr/local/bin/manifest-tool https://github.com/estesp/manifest-tool/releases/download/v0.7.0/manifest-tool-linux-$(IMAGE_ARCH)
	sudo chmod +x /usr/local/bin/manifest-tool
.PHONY: docker-manifest-tool

docker-push-manifests: docker-manifest-tool
	cp -f manifest.yaml /tmp/manifest-flex.yaml
	cp -f manifest.yaml /tmp/manifest-provisioner.yaml
	# Replace the variables in the template with the ones for our flex and provisioner images
	sed -i -e "s|__RELEASE_TAG__|$(DRIVER_VERSION)|g" -e "s|__IMAGE_NAME__|$(IMAGE_FLEXPLUGIN)|g" -e "s|__IMAGE_REPO__|$(IMAGE_REPO)|g" /tmp/manifest-flex.yaml
	sed -i -e "s|__RELEASE_TAG__|$(DRIVER_VERSION)|g" -e "s|__IMAGE_NAME__|$(IMAGE_PROVISONER)|g" -e "s|__IMAGE_REPO__|$(IMAGE_REPO)|g" /tmp/manifest-provisioner.yaml
	# Use the Manifest tool to push the manifest lists to the docker registry for the multi-arch image support
	manifest-tool push from-spec /tmp/manifest-flex.yaml
	manifest-tool push from-spec /tmp/manifest-provisioner.yaml
.PHONY: docker-push-manifests
