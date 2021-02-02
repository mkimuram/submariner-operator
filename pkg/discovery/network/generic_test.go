/*
© 2019 Red Hat, Inc. and others.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package network

import (
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	fakecorev1 "k8s.io/client-go/kubernetes/typed/core/v1/fake"
	"k8s.io/client-go/testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const testPodCIDR = "1.2.3.4/16"
const testServiceCIDR = "4.5.6.7/16"

var errValid = fmt.Errorf("The Service \"tst\" is invalid: spec.clusterIPs: Invalid value: []string{\"1.1.1.1\"}: failed to allocated ip:1.1.1.1 with error:provided IP is not in the valid range. The range of valid IPs is %s", testServiceCIDR)
var errInvalid = fmt.Errorf("%s", testServiceCIDR)

var _ = Describe("discoverGenericNetwork", func() {
	When("There is no node", func() {
		It("Should return nil cluster network", func() {
			clusterNet, err := testDiscoverGenericWith(
				errValid,
			)
			Expect(err).To(HaveOccurred())
			Expect(clusterNet).To(BeNil())
		})
	})

	When("There is no pod cidr information on node", func() {
		It("Should return nil cluster network", func() {
			clusterNet, err := testDiscoverGenericWith(
				errValid,
				fakeNode("node1", ""),
				fakeNode("node2", ""),
			)
			Expect(err).To(HaveOccurred())
			Expect(clusterNet).To(BeNil())
		})
	})

	When("There is pod cidr information on any of nodes, but invalid service creation doesn't fail", func() {
		It("Should return nil cluster network", func() {
			clusterNet, err := testDiscoverGenericWith(
				nil,
				fakeNode("node1", ""),
				fakeNode("node2", testPodCIDR),
			)
			Expect(err).To(HaveOccurred())
			Expect(clusterNet).To(BeNil())
		})
	})

	When("There is pod cidr information on any of nodes, but invalid service creation fails with invalid message", func() {
		var clusterNet *ClusterNetwork
		var err error

		BeforeEach(func() {
			clusterNet, err = testDiscoverGenericWith(
				errInvalid,
				fakeNode("node1", testPodCIDR),
			)
			Expect(err).To(HaveOccurred())
			Expect(clusterNet).To(BeNil())
		})
	})

	When("There is pod cidr information on any of nodes, and invalid service creation fails properly", func() {
		var clusterNet *ClusterNetwork
		var err error

		BeforeEach(func() {
			clusterNet, err = testDiscoverGenericWith(
				errValid,
				fakeNode("node1", testPodCIDR),
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(clusterNet).NotTo(BeNil())
		})

		It("Should return the ClusterNetwork structure with PodCIDR", func() {
			Expect(clusterNet.PodCIDRs).To(Equal([]string{testPodCIDR}))
		})

		It("Should identify the networkplugin as generic", func() {
			Expect(clusterNet.NetworkPlugin).To(BeIdenticalTo("generic"))
		})

		It("Should return the ClusterNetwork structure with service CIDR", func() {
			Expect(clusterNet.ServiceCIDRs).To(Equal([]string{testServiceCIDR}))
		})
	})

})

func testDiscoverGenericWith(expectedErr error, objects ...runtime.Object) (*ClusterNetwork, error) {
	clientSet := fake.NewSimpleClientset(objects...)
	if expectedErr != nil {
		clientSet.CoreV1().(*fakecorev1.FakeCoreV1).PrependReactor("create", "services", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
			return true, nil, expectedErr
		})
	} else {
		clientSet.CoreV1().(*fakecorev1.FakeCoreV1).PrependReactor("create", "services", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
			return true, &v1.Service{}, nil
		})
	}
	return discoverGenericNetwork(clientSet)
}
