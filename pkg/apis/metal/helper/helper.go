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
	"encoding/json"
	"fmt"

	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"
)

// FindMachineImage takes a list of machine images and tries to find the first entry
// whose name, version, and zone matches with the given name, version, and zone. If no such entry is
// found then an error will be returned.
func FindMachineImage(machineImages []metal.MachineImage, name, version string) (*metal.MachineImage, error) {
	for _, machineImage := range machineImages {
		if machineImage.Name == name && machineImage.Version == version {
			return &machineImage, nil
		}
	}
	return nil, fmt.Errorf("no machine image with name %q, version %q found", name, version)
}

// MergeIAMConfig merges the one iam config into the other
func MergeIAMConfig(into *metal.IAMConfig, from *metal.IAMConfig) (*metal.IAMConfig, error) {
	if into == nil && from == nil {
		return nil, nil
	}

	if from == nil {
		copy := *into
		return &copy, nil
	}

	if into == nil {
		copy := *from
		return &copy, nil
	}

	merged := *into
	tmp, err := json.Marshal(from)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(tmp, &merged)
	if err != nil {
		return nil, err
	}
	return &merged, nil
}

// FindMetalControlPlane returns the metal control plane from a given region name
func FindMetalControlPlane(cloudProfileConfig *metal.CloudProfileConfig, region string) (*metal.MetalControlPlane, *metal.Partition, error) {
	for _, mcp := range cloudProfileConfig.MetalControlPlanes {
		for partitionName, p := range mcp.Partitions {
			if partitionName == region {
				return &mcp, &p, nil
			}
		}
	}
	return nil, nil, fmt.Errorf("no metal control plane found for region %s in cloud profile config", region)
}
