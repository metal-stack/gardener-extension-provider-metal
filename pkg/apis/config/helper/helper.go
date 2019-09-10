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

// FindImage takes a list of machine images, and the desired image name and version. It tries
// to find the image with the given name and version. If it cannot be found then an error
// is returned.
func FindImage(machineImages []config.MachineImage, imageName, version string) (string, error) {
	for _, machineImage := range machineImages {
		if machineImage.Name == imageName && machineImage.Version == version {
			return machineImage.Image, nil
		}
	}

	return "", fmt.Errorf("could not find an image for name %q in version %q", imageName, version)
}
