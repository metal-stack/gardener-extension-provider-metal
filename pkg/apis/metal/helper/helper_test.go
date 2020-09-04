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
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"
)

func TestMergeIAMConfig(t *testing.T) {
	type args struct {
		into *metal.IAMConfig
		from *metal.IAMConfig
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    *metal.IAMConfig
	}{
		{
			name: "nil stays nil",
			args: args{
				into: nil,
				from: nil,
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "do not alter anything if from is nil",
			args: args{
				into: &metal.IAMConfig{IdmConfig: &metal.IDMConfig{Idmtype: "a"}},
				from: nil,
			},
			want:    &metal.IAMConfig{IdmConfig: &metal.IDMConfig{Idmtype: "a"}},
			wantErr: false,
		},
		{
			name: "set into to from if into is nil",
			args: args{
				into: nil,
				from: &metal.IAMConfig{IdmConfig: &metal.IDMConfig{Idmtype: "a"}},
			},
			want:    &metal.IAMConfig{IdmConfig: &metal.IDMConfig{Idmtype: "a"}},
			wantErr: false,
		},
		{
			name: "merge field in from into into",
			args: args{
				into: &metal.IAMConfig{IdmConfig: &metal.IDMConfig{Idmtype: "a"}},
				from: &metal.IAMConfig{IssuerConfig: &metal.IssuerConfig{Url: "url"}},
			},
			want:    &metal.IAMConfig{IdmConfig: &metal.IDMConfig{Idmtype: "a"}, IssuerConfig: &metal.IssuerConfig{Url: "url"}},
			wantErr: false,
		},
		{
			name: "from overrides into",
			args: args{
				into: &metal.IAMConfig{IdmConfig: &metal.IDMConfig{Idmtype: "a"}},
				from: &metal.IAMConfig{IdmConfig: &metal.IDMConfig{Idmtype: "b"}},
			},
			want:    &metal.IAMConfig{IdmConfig: &metal.IDMConfig{Idmtype: "b"}},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MergeIAMConfig(tt.args.into, tt.args.from)
			if (err != nil) != tt.wantErr {
				t.Errorf("MergeIAMConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("MergeIAMConfig() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
