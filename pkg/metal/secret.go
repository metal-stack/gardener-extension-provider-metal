// Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package metal

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

// ReadCredentialsSecret reads a secret containing credentials.
func ReadCredentialsSecret(secret *corev1.Secret) (*Credentials, error) {
	if secret.Data == nil {
		return nil, fmt.Errorf("secret does not contain any data")
	}

	url, ok := secret.Data[MetalAPIURL]
	if !ok {
		return nil, fmt.Errorf("missing %q field in secret", MetalAPIURL)
	}

	hmac, ok := secret.Data[MetalAPIHMac]
	if !ok {
		return nil, fmt.Errorf("missing %q field in secret", MetalAPIHMac)
	}

	key, ok := secret.Data[MetalAPIKey]
	if !ok {
		return nil, fmt.Errorf("missing %q field in secret", MetalAPIKey)
	}

	return &Credentials{
		MetalAPIURL:  url,
		MetalAPIHMac: hmac,
		MetalAPIKey:  key,
	}, nil
}
