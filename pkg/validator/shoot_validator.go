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

package validator

import (
	"context"
	"errors"
	"reflect"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/gardener/gardener/pkg/apis/garden"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"
	"github.com/metal-pod/gardener-extension-provider-metal/pkg/apis/metal/helper"
	metalvalidation "github.com/metal-pod/gardener-extension-provider-metal/pkg/apis/metal/validation"

	"k8s.io/apimachinery/pkg/util/validation/field"
)

func (v *Shoot) validateShoot(ctx context.Context, shoot *garden.Shoot) error {
	// Provider validation
	fldPath := field.NewPath("spec", "provider")

	// InfrastructureConfig
	infraConfigFldPath := fldPath.Child("infrastructureConfig")

	if shoot.Spec.Provider.InfrastructureConfig == nil {
		return field.Required(infraConfigFldPath, "InfrastructureConfig must be set for metal shoots")
	}

	infraConfig, err := decodeInfrastructureConfig(v.decoder, shoot.Spec.Provider.InfrastructureConfig, infraConfigFldPath)
	if err != nil {
		return err
	}

	if errList := metalvalidation.ValidateInfrastructureConfig(infraConfig); len(errList) != 0 {
		return errList.ToAggregate()
	}

	cloudProfile := &gardencorev1beta1.CloudProfile{}
	if err := v.client.Get(ctx, kutil.Key(shoot.Spec.CloudProfileName), cloudProfile); err != nil {
		return err
	}

	v.Logger.Info("got cloud profile", "provider config", cloudProfile.Spec.ProviderConfig)
	v.Logger.Info("got cloud profile", "provider config raw", cloudProfile.Spec.ProviderConfig.Raw)

	cloudProfileConfig, err := helper.DecodeCloudProfileConfig(cloudProfile)
	if err != nil {
		return err
	}

	if errList := metalvalidation.ValidateInfrastructureConfigAgainstCloudProfile(infraConfig, shoot, cloudProfile, cloudProfileConfig, infraConfigFldPath); len(errList) != 0 {
		return errList.ToAggregate()
	}

	// ControlPlaneConfig
	controlPlaneConfigFldPath := fldPath.Child("controlPlaneConfig")

	if shoot.Spec.Provider.ControlPlaneConfig == nil {
		return field.Required(infraConfigFldPath, "ControlPlaneConfig must be set for metal shoots")
	}

	controlPlaneConfig, err := decodeControlPlaneConfig(v.decoder, shoot.Spec.Provider.ControlPlaneConfig, fldPath.Child("controlPlaneConfig"))
	if err != nil {
		return err
	}

	controlPlaneConfig.IAMConfig, err = helper.MergeIAMConfig(controlPlaneConfig.IAMConfig, cloudProfileConfig.IAMConfig)
	if err != nil {
		return err
	}

	if errList := metalvalidation.ValidateControlPlaneConfig(controlPlaneConfig, cloudProfile, controlPlaneConfigFldPath); len(errList) != 0 {
		return errList.ToAggregate()
	}

	// Shoot workers
	if errList := metalvalidation.ValidateWorkers(shoot.Spec.Provider.Workers, cloudProfile, fldPath); len(errList) != 0 {
		return errList.ToAggregate()
	}

	return nil
}

func (v *Shoot) validateShootUpdate(ctx context.Context, oldShoot, shoot *garden.Shoot) error {
	fldPath := field.NewPath("spec", "provider")

	// InfrastructureConfig update
	if shoot.Spec.Provider.InfrastructureConfig == nil {
		return field.Required(fldPath.Child("infrastructureConfig"), "InfrastructureConfig must be set for metal shoots")
	}

	infraConfig, err := decodeInfrastructureConfig(v.decoder, shoot.Spec.Provider.InfrastructureConfig, fldPath)
	if err != nil {
		return err
	}

	if oldShoot.Spec.Provider.InfrastructureConfig == nil {
		return field.InternalError(fldPath.Child("infrastructureConfig"), errors.New("InfrastructureConfig is not available on old shoot"))
	}

	oldInfraConfig, err := decodeInfrastructureConfig(v.decoder, oldShoot.Spec.Provider.InfrastructureConfig, fldPath)
	if err != nil {
		return err
	}

	if !reflect.DeepEqual(oldInfraConfig, infraConfig) {
		if errList := metalvalidation.ValidateInfrastructureConfigUpdate(oldInfraConfig, infraConfig); len(errList) != 0 {
			return errList.ToAggregate()
		}
	}

	return v.validateShoot(ctx, shoot)
}
