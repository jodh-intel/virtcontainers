//
// Copyright (c) 2016 Intel Corporation
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
//

package virtcontainers

import (
	"net"
	"os"
	"reflect"
	"testing"
)

func testNetworkModelSet(t *testing.T, value string, expected NetworkModel) {
	var netModel NetworkModel

	err := netModel.Set(value)
	if err != nil {
		t.Fatal(err)
	}

	if netModel != expected {
		t.Fatal()
	}
}

func TestNoopNetworkModelSet(t *testing.T) {
	testNetworkModelSet(t, "noop", NoopNetworkModel)
}

func TestCNINetworkModelSet(t *testing.T) {
	testNetworkModelSet(t, "CNI", CNINetworkModel)
}

func TestCNMNetworkModelSet(t *testing.T) {
	testNetworkModelSet(t, "CNM", CNMNetworkModel)
}

func TestNetworkModelSetFailure(t *testing.T) {
	var netModel NetworkModel

	err := netModel.Set("wrong-value")
	if err == nil {
		t.Fatal(err)
	}
}

func testNetworkModelString(t *testing.T, netModel *NetworkModel, expected string) {
	result := netModel.String()

	if result != expected {
		t.Fatal()
	}
}

func TestNoopNetworkModelString(t *testing.T) {
	netModel := NoopNetworkModel
	testNetworkModelString(t, &netModel, string(NoopNetworkModel))
}

func TestCNINetworkModelString(t *testing.T) {
	netModel := CNINetworkModel
	testNetworkModelString(t, &netModel, string(CNINetworkModel))
}

func TestCNMNetworkModelString(t *testing.T) {
	netModel := CNMNetworkModel
	testNetworkModelString(t, &netModel, string(CNMNetworkModel))
}

func TestWrongNetworkModelString(t *testing.T) {
	var netModel NetworkModel
	testNetworkModelString(t, &netModel, "")
}

func testNewNetworkFromNetworkModel(t *testing.T, netModel NetworkModel, expected interface{}) {
	result := newNetwork(netModel)

	if reflect.DeepEqual(result, expected) == false {
		t.Fatal()
	}
}

func TestNewNoopNetworkFromNetworkModel(t *testing.T) {
	testNewNetworkFromNetworkModel(t, NoopNetworkModel, &noopNetwork{})
}

func TestNewCNINetworkFromNetworkModel(t *testing.T) {
	testNewNetworkFromNetworkModel(t, CNINetworkModel, &cni{})
}

func TestNewCNMNetworkFromNetworkModel(t *testing.T) {
	testNewNetworkFromNetworkModel(t, CNMNetworkModel, &cnm{})
}

func TestNewUnknownNetworkFromNetworkModel(t *testing.T) {
	var netModel NetworkModel
	testNewNetworkFromNetworkModel(t, netModel, &noopNetwork{})
}

func TestCreateDeleteNetNS(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip(testDisabledAsNonRoot)
	}

	netNSPath, err := createNetNS()
	if err != nil {
		t.Fatal(err)
	}

	if netNSPath == "" {
		t.Fatal()
	}

	_, err = os.Stat(netNSPath)
	if err != nil {
		t.Fatal(err)
	}

	err = deleteNetNS(netNSPath, true)
	if err != nil {
		t.Fatal(err)
	}
}

func TestCreateNetworkEndpoint(t *testing.T) {
	macAddr := net.HardwareAddr{0x02, 0x00, 0xCA, 0xFE, 0x00, 0x04}

	expected := Endpoint{
		NetPair: NetworkInterfacePair{
			ID:   "uniqueTestID-4",
			Name: "br4",
			VirtIface: NetworkInterface{
				Name:     "eth4",
				HardAddr: macAddr.String(),
			},
			TAPIface: NetworkInterface{
				Name: "tap4",
			},
		},
	}

	result, err := createNetworkEndpoint(4, "uniqueTestID", "")
	if err != nil {
		t.Fatal(err)
	}

	if reflect.DeepEqual(result, expected) == false {
		t.Fatal()
	}
}

func TestCreateNetworkEndpointChooseIfaceName(t *testing.T) {
	macAddr := net.HardwareAddr{0x02, 0x00, 0xCA, 0xFE, 0x00, 0x04}

	expected := Endpoint{
		NetPair: NetworkInterfacePair{
			ID:   "uniqueTestID-4",
			Name: "br4",
			VirtIface: NetworkInterface{
				Name:     "eth1",
				HardAddr: macAddr.String(),
			},
			TAPIface: NetworkInterface{
				Name: "tap4",
			},
		},
	}

	result, err := createNetworkEndpoint(4, "uniqueTestID", "eth1")
	if err != nil {
		t.Fatal(err)
	}

	if reflect.DeepEqual(result, expected) == false {
		t.Fatal()
	}
}

func TestCreateNetworkEndpointInvalidArgs(t *testing.T) {
	type endpointValues struct {
		idx      int
		uniqueID string
		ifName   string
	}

	// all elements are expected to result in failure
	failingValues := []endpointValues{
		{-1, "foo", "bar"},
		{-1, "foo", ""},
		{-3, "foo", "bar"},
		{-3, "foo", ""},
		{0, "", "bar"},
		{0, "", ""},
		{1, "", "bar"},
		{1, "", ""},
	}

	for _, d := range failingValues {
		result, err := createNetworkEndpoint(d.idx, d.uniqueID, d.ifName)
		if err == nil {
			t.Fatalf("expected invalid endpoint for %v, got %v", d, result)
		}
	}
}

func TestCreateNetworkEndpoints(t *testing.T) {
	numOfEndpoints := 3

	endpoints, err := createNetworkEndpoints(numOfEndpoints)
	if err != nil {
		t.Fatal(err)
	}

	if len(endpoints) != numOfEndpoints {
		t.Fatal()
	}
}

func TestCreateNetworkEndpointsFailure(t *testing.T) {
	_, err := createNetworkEndpoints(0)
	if err == nil {
		t.Fatal(err)
	}
}
