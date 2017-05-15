# Copyright (c) 2017 Intel Corporation

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
#

#!/bin/bash

set -e
set -x

go_packages=$(go list ./... | grep -v vendor)

sudo -E go test $go_packages -cover

sudo -E go test -bench=.

sudo -E go test -bench=CreateStartStopDeletePodQemuHypervisorNoopAgentNetworkCNI -benchtime=60s

sudo -E go test -bench=CreateStartStopDeletePodQemuHypervisorHyperstartAgentNetworkCNI -benchtime=60s
