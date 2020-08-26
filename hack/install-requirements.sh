#!/bin/bash
#
# Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
set -e

DIRNAME="$(echo "$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )")"

cd "$DIRNAME/../tools"
export GO111MODULE=on
echo "Installing requirements"
go install "github.com/gobuffalo/packr/v2/packr2"
go install "github.com/onsi/ginkgo/ginkgo"
go install "github.com/golang/mock/mockgen"
go install "github.com/golangci/golangci-lint/cmd/golangci-lint"
curl -s "https://raw.githubusercontent.com/helm/helm/v2.16.9/scripts/get" | bash -s -- --version 'v2.13.1'

if [[ "$(uname -s)" == *"Darwin"* ]]; then
  cat <<EOM
You are running in a MAC OS environment!

Please make sure you have installed the following requirements:

- GNU Core Utils
- GNU Tar

Brew command:
$ brew install coreutils gnu-tar

Please allow them to be used without their "g" prefix:
$ export PATH=/usr/local/opt/coreutils/libexec/gnubin:\$PATH

EOM
fi

