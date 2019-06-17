// Copyright (c) 2018 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package helper

import (
	"fmt"

	"github.com/metal-pod/gardener-extension-provider-metal/pkg/apis/config"
)

// FindImageID TODO: write docs takes a list of machine images, and the desired image name, version, and region. It tries
// to find the image with the given name and version in the desired region. If it cannot be found then an error
// is returned.
func FindImageID(machineImages []config.MachineImage, imageName, version string) (string, error) {
	for _, machineImage := range machineImages {
		if machineImage.Name != imageName || machineImage.Version != version {
			continue
		}
		return machineImage.ImageID, nil
	}

	return "", fmt.Errorf("could not find an image id for machine image %q in version %q", imageName, version)
}
