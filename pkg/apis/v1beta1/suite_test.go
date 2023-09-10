/*
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

package v1beta1_test

import (
	"context"
	"strings"
	"testing"

	"github.com/Pallinder/go-randomdata"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "knative.dev/pkg/logging/testing"
	"knative.dev/pkg/ptr"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/aws/aws-sdk-go/aws"

	"github.com/aws/karpenter/pkg/apis/v1alpha1"
	"github.com/aws/karpenter/pkg/apis/v1beta1"
	"github.com/aws/karpenter/pkg/test"
)

var ctx context.Context

func TestAPIs(t *testing.T) {
	ctx = TestContextWithLogger(t)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Validation")
}

var _ = Describe("Validation", func() {
	var nc *v1beta1.NodeClass

	BeforeEach(func() {
		nc = &v1beta1.NodeClass{
			ObjectMeta: metav1.ObjectMeta{Name: strings.ToLower(randomdata.SillyName())},
			Spec: v1beta1.NodeClassSpec{
				SubnetSelectorTerms: []v1beta1.SubnetSelectorTerm{
					{
						Tags: map[string]string{
							"foo": "bar",
						},
					},
				},
				SecurityGroupSelectorTerms: []v1beta1.SecurityGroupSelectorTerm{
					{
						Tags: map[string]string{
							"foo": "bar",
						},
					},
				},
			},
		}
	})

	Context("UserData", func() {
		It("should succeed if user data is empty", func() {
			Expect(nc.Validate(ctx)).To(Succeed())
		})
		It("should fail if Windows2019 AMIFamily is specified", func() {
			nc.Spec.AMIFamily = &v1alpha1.AMIFamilyWindows2019
			nc.Spec.UserData = ptr.String("someUserData")
			Expect(nc.Validate(ctx)).To(Not(Succeed()))
		})
		It("should fail if Windows2022 AMIFamily is specified", func() {
			nc.Spec.AMIFamily = &v1alpha1.AMIFamilyWindows2022
			nc.Spec.UserData = ptr.String("someUserData")
			Expect(nc.Validate(ctx)).To(Not(Succeed()))
		})
	})
	Context("Tags", func() {
		It("should succeed when tags are empty", func() {
			nc.Spec.Tags = map[string]string{}
			Expect(nc.Validate(ctx)).To(Succeed())
		})
		It("should succeed if tags aren't in restricted tag keys", func() {
			nc.Spec.Tags = map[string]string{
				"karpenter.sh/custom-key": "value",
				"karpenter.sh/managed":    "true",
				"kubernetes.io/role/key":  "value",
			}
			Expect(nc.Validate(ctx)).To(Succeed())
		})
		It("should succeed by validating that regex is properly escaped", func() {
			nc.Spec.Tags = map[string]string{
				"karpenterzsh/provisioner-name": "value",
			}
			Expect(nc.Validate(ctx)).To(Succeed())
			nc.Spec.Tags = map[string]string{
				"kubernetesbio/cluster/test": "value",
			}
			Expect(nc.Validate(ctx)).To(Succeed())
			nc.Spec.Tags = map[string]string{
				"karpenterzsh/managed-by": "test",
			}
			Expect(nc.Validate(ctx)).To(Succeed())
		})
		It("should fail if tags contain a restricted domain key", func() {
			nc.Spec.Tags = map[string]string{
				"karpenter.sh/provisioner-name": "value",
			}
			Expect(nc.Validate(ctx)).To(Not(Succeed()))
			nc.Spec.Tags = map[string]string{
				"kubernetes.io/cluster/test": "value",
			}
			Expect(nc.Validate(ctx)).To(Not(Succeed()))
			nc.Spec.Tags = map[string]string{
				"karpenter.sh/managed-by": "test",
			}
			Expect(nc.Validate(ctx)).To(Not(Succeed()))
		})
	})
	Context("SubnetSelectorTerms", func() {
		It("should succeed with a valid subnet selector on tags", func() {
			nc.Spec.SubnetSelectorTerms = []v1beta1.SubnetSelectorTerm{
				{
					Tags: map[string]string{
						"test": "testvalue",
					},
				},
			}
			Expect(nc.Validate(ctx)).To(Succeed())
		})
		It("should succeed with a valid subnet selector on id", func() {
			nc.Spec.SubnetSelectorTerms = []v1beta1.SubnetSelectorTerm{
				{
					ID: "subnet-12345749",
				},
			}
			Expect(nc.Validate(ctx)).To(Succeed())
		})
		It("should fail when subnet selector terms is set to nil", func() {
			nc.Spec.SubnetSelectorTerms = nil
			Expect(nc.Validate(ctx)).ToNot(Succeed())
		})
		It("should fail when no subnet selector terms exist", func() {
			nc.Spec.SubnetSelectorTerms = []v1beta1.SubnetSelectorTerm{}
			Expect(nc.Validate(ctx)).ToNot(Succeed())
		})
		It("should fail when a subnet selector term has no values", func() {
			nc.Spec.SubnetSelectorTerms = []v1beta1.SubnetSelectorTerm{
				{},
			}
			Expect(nc.Validate(ctx)).ToNot(Succeed())
		})
		It("should fail when a subnet selector term has no tag map values", func() {
			nc.Spec.SubnetSelectorTerms = []v1beta1.SubnetSelectorTerm{
				{
					Tags: map[string]string{},
				},
			}
			Expect(nc.Validate(ctx)).ToNot(Succeed())
		})
		It("should fail when a subnet selector term has a tag map key that is empty", func() {
			nc.Spec.SubnetSelectorTerms = []v1beta1.SubnetSelectorTerm{
				{
					Tags: map[string]string{
						"test": "",
					},
				},
			}
			Expect(nc.Validate(ctx)).ToNot(Succeed())
		})
		It("should fail when a subnet selector term has a tag map value that is empty", func() {
			nc.Spec.SubnetSelectorTerms = []v1beta1.SubnetSelectorTerm{
				{
					Tags: map[string]string{
						"": "testvalue",
					},
				},
			}
			Expect(nc.Validate(ctx)).ToNot(Succeed())
		})
		It("should fail when the last subnet selector is invalid", func() {
			nc.Spec.SubnetSelectorTerms = []v1beta1.SubnetSelectorTerm{
				{
					Tags: map[string]string{
						"test": "testvalue",
					},
				},
				{
					Tags: map[string]string{
						"test2": "testvalue2",
					},
				},
				{
					Tags: map[string]string{
						"test3": "testvalue3",
					},
				},
				{
					Tags: map[string]string{
						"": "testvalue4",
					},
				},
			}
			Expect(nc.Validate(ctx)).ToNot(Succeed())
		})
		It("should fail when specifying id with tags", func() {
			nc.Spec.SubnetSelectorTerms = []v1beta1.SubnetSelectorTerm{
				{
					ID: "subnet-12345749",
					Tags: map[string]string{
						"test": "testvalue",
					},
				},
			}
			Expect(nc.Validate(ctx)).ToNot(Succeed())
		})
	})
	Context("SecurityGroupSelectorTerms", func() {
		It("should succeed with a valid security group selector on tags", func() {
			nc.Spec.SecurityGroupSelectorTerms = []v1beta1.SecurityGroupSelectorTerm{
				{
					Tags: map[string]string{
						"test": "testvalue",
					},
				},
			}
			Expect(nc.Validate(ctx)).To(Succeed())
		})
		It("should succeed with a valid security group selector on id", func() {
			nc.Spec.SecurityGroupSelectorTerms = []v1beta1.SecurityGroupSelectorTerm{
				{
					ID: "sg-12345749",
				},
			}
			Expect(nc.Validate(ctx)).To(Succeed())
		})
		It("should succeed with a valid security group selector on name", func() {
			nc.Spec.SecurityGroupSelectorTerms = []v1beta1.SecurityGroupSelectorTerm{
				{
					Name: "testname",
				},
			}
			Expect(nc.Validate(ctx)).To(Succeed())
		})
		It("should fail when security group selector terms is set to nil", func() {
			nc.Spec.SecurityGroupSelectorTerms = nil
			Expect(nc.Validate(ctx)).ToNot(Succeed())
		})
		It("should fail when no security group selector terms exist", func() {
			nc.Spec.SecurityGroupSelectorTerms = []v1beta1.SecurityGroupSelectorTerm{}
			Expect(nc.Validate(ctx)).ToNot(Succeed())
		})
		It("should fail when a security group selector term has no values", func() {
			nc.Spec.SecurityGroupSelectorTerms = []v1beta1.SecurityGroupSelectorTerm{
				{},
			}
			Expect(nc.Validate(ctx)).ToNot(Succeed())
		})
		It("should fail when a security group selector term has no tag map values", func() {
			nc.Spec.SecurityGroupSelectorTerms = []v1beta1.SecurityGroupSelectorTerm{
				{
					Tags: map[string]string{},
				},
			}
			Expect(nc.Validate(ctx)).ToNot(Succeed())
		})
		It("should fail when a security group selector term has a tag map key that is empty", func() {
			nc.Spec.SecurityGroupSelectorTerms = []v1beta1.SecurityGroupSelectorTerm{
				{
					Tags: map[string]string{
						"test": "",
					},
				},
			}
			Expect(nc.Validate(ctx)).ToNot(Succeed())
		})
		It("should fail when a security group selector term has a tag map value that is empty", func() {
			nc.Spec.SecurityGroupSelectorTerms = []v1beta1.SecurityGroupSelectorTerm{
				{
					Tags: map[string]string{
						"": "testvalue",
					},
				},
			}
			Expect(nc.Validate(ctx)).ToNot(Succeed())
		})
		It("should fail when the last security group selector is invalid", func() {
			nc.Spec.SecurityGroupSelectorTerms = []v1beta1.SecurityGroupSelectorTerm{
				{
					Tags: map[string]string{
						"test": "testvalue",
					},
				},
				{
					Tags: map[string]string{
						"test2": "testvalue2",
					},
				},
				{
					Tags: map[string]string{
						"test3": "testvalue3",
					},
				},
				{
					Tags: map[string]string{
						"": "testvalue4",
					},
				},
			}
			Expect(nc.Validate(ctx)).ToNot(Succeed())
		})
		It("should fail when specifying id with tags", func() {
			nc.Spec.SecurityGroupSelectorTerms = []v1beta1.SecurityGroupSelectorTerm{
				{
					ID: "sg-12345749",
					Tags: map[string]string{
						"test": "testvalue",
					},
				},
			}
			Expect(nc.Validate(ctx)).ToNot(Succeed())
		})
		It("should fail when specifying id with name", func() {
			nc.Spec.SecurityGroupSelectorTerms = []v1beta1.SecurityGroupSelectorTerm{
				{
					ID:   "sg-12345749",
					Name: "my-security-group",
				},
			}
			Expect(nc.Validate(ctx)).ToNot(Succeed())
		})
		It("should fail when specifying name with tags", func() {
			nc.Spec.SecurityGroupSelectorTerms = []v1beta1.SecurityGroupSelectorTerm{
				{
					Name: "my-security-group",
					Tags: map[string]string{
						"test": "testvalue",
					},
				},
			}
			Expect(nc.Validate(ctx)).ToNot(Succeed())
		})
	})
	Context("AMISelectorTerms", func() {
		It("should succeed with a valid ami selector on tags", func() {
			nc.Spec.AMISelectorTerms = []v1beta1.AMISelectorTerm{
				{
					Tags: map[string]string{
						"test": "testvalue",
					},
				},
			}
			Expect(nc.Validate(ctx)).To(Succeed())
		})
		It("should succeed with a valid ami selector on id", func() {
			nc.Spec.AMISelectorTerms = []v1beta1.AMISelectorTerm{
				{
					ID: "sg-12345749",
				},
			}
			Expect(nc.Validate(ctx)).To(Succeed())
		})
		It("should succeed with a valid ami selector on name", func() {
			nc.Spec.AMISelectorTerms = []v1beta1.AMISelectorTerm{
				{
					Name: "testname",
				},
			}
			Expect(nc.Validate(ctx)).To(Succeed())
		})
		It("should succeed with a valid ami selector on ssm", func() {
			nc.Spec.AMISelectorTerms = []v1beta1.AMISelectorTerm{
				{
					SSM: "/test/ssm/path",
				},
			}
			Expect(nc.Validate(ctx)).To(Succeed())
		})
		It("should fail when a ami selector term has no values", func() {
			nc.Spec.AMISelectorTerms = []v1beta1.AMISelectorTerm{
				{},
			}
			Expect(nc.Validate(ctx)).ToNot(Succeed())
		})
		It("should fail when a ami selector term has no tag map values", func() {
			nc.Spec.AMISelectorTerms = []v1beta1.AMISelectorTerm{
				{
					Tags: map[string]string{},
				},
			}
			Expect(nc.Validate(ctx)).ToNot(Succeed())
		})
		It("should fail when a ami selector term has a tag map key that is empty", func() {
			nc.Spec.AMISelectorTerms = []v1beta1.AMISelectorTerm{
				{
					Tags: map[string]string{
						"test": "",
					},
				},
			}
			Expect(nc.Validate(ctx)).ToNot(Succeed())
		})
		It("should fail when a ami selector term has a tag map value that is empty", func() {
			nc.Spec.AMISelectorTerms = []v1beta1.AMISelectorTerm{
				{
					Tags: map[string]string{
						"": "testvalue",
					},
				},
			}
			Expect(nc.Validate(ctx)).ToNot(Succeed())
		})
		It("should fail when the last ami selector is invalid", func() {
			nc.Spec.AMISelectorTerms = []v1beta1.AMISelectorTerm{
				{
					Tags: map[string]string{
						"test": "testvalue",
					},
				},
				{
					Tags: map[string]string{
						"test2": "testvalue2",
					},
				},
				{
					Tags: map[string]string{
						"test3": "testvalue3",
					},
				},
				{
					Tags: map[string]string{
						"": "testvalue4",
					},
				},
			}
			Expect(nc.Validate(ctx)).ToNot(Succeed())
		})
		It("should fail when specifying id with tags", func() {
			nc.Spec.AMISelectorTerms = []v1beta1.AMISelectorTerm{
				{
					ID: "ami-12345749",
					Tags: map[string]string{
						"test": "testvalue",
					},
				},
			}
			Expect(nc.Validate(ctx)).ToNot(Succeed())
		})
		It("should fail when specifying id with name", func() {
			nc.Spec.AMISelectorTerms = []v1beta1.AMISelectorTerm{
				{
					ID:   "ami-12345749",
					Name: "my-custom-ami",
				},
			}
			Expect(nc.Validate(ctx)).ToNot(Succeed())
		})
		It("should fail when specifying id with owner", func() {
			nc.Spec.AMISelectorTerms = []v1beta1.AMISelectorTerm{
				{
					ID:    "ami-12345749",
					Owner: "123456789",
				},
			}
			Expect(nc.Validate(ctx)).ToNot(Succeed())
		})
		It("should fail when specifying id with ssm", func() {
			nc.Spec.AMISelectorTerms = []v1beta1.AMISelectorTerm{
				{
					ID:  "ami-12345749",
					SSM: "/test/ssm/path",
				},
			}
			Expect(nc.Validate(ctx)).ToNot(Succeed())
		})
	})
	Context("NodeClass Hash", func() {
		var nodeClass *v1beta1.NodeClass
		BeforeEach(func() {
			nodeClass = test.NodeClass(v1beta1.NodeClass{
				Spec: v1beta1.NodeClassSpec{
					AMIFamily: aws.String(v1alpha1.AMIFamilyAL2),
					Context:   aws.String("context-1"),
					Role:      aws.String("role-1"),
					Tags: map[string]string{
						"keyTag-1": "valueTag-1",
						"keyTag-2": "valueTag-2",
					},
					MetadataOptions: &v1beta1.MetadataOptions{
						HTTPEndpoint: aws.String("test-metadata-1"),
					},
					BlockDeviceMappings: []*v1beta1.BlockDeviceMapping{
						{
							DeviceName: aws.String("map-device-1"),
						},
						{
							DeviceName: aws.String("map-device-2"),
						},
					},
					UserData:           aws.String("userdata-test-1"),
					DetailedMonitoring: aws.Bool(false),
				},
			})
		})
		DescribeTable("should change hash when static fields are updated", func(nodeClassSpec v1beta1.NodeClassSpec) {
			hash := nodeClass.Hash()
			nodeClass.Spec = nodeClassSpec
			updatedHash := nodeClass.Hash()
			Expect(hash).ToNot(Equal(updatedHash))
		},
			Entry("InstanceProfile Drift", v1beta1.NodeClassSpec{Role: aws.String("role-2")}),
			Entry("UserData Drift", v1beta1.NodeClassSpec{UserData: aws.String("userdata-test-2")}),
			Entry("Tags Drift", v1beta1.NodeClassSpec{Tags: map[string]string{"keyTag-test-3": "valueTag-test-3"}}),
			Entry("MetadataOptions Drift", v1beta1.NodeClassSpec{MetadataOptions: &v1beta1.MetadataOptions{HTTPEndpoint: aws.String("test-metadata-2")}}),
			Entry("BlockDeviceMappings Drift", v1beta1.NodeClassSpec{BlockDeviceMappings: []*v1beta1.BlockDeviceMapping{{DeviceName: aws.String("map-device-test-3")}}}),
			Entry("Context Drift", v1beta1.NodeClassSpec{Context: aws.String("context-2")}),
			Entry("DetailedMonitoring Drift", v1beta1.NodeClassSpec{DetailedMonitoring: aws.Bool(true)}),
			Entry("AMIFamily Drift", v1beta1.NodeClassSpec{AMIFamily: aws.String(v1alpha1.AMIFamilyBottlerocket)}),
			Entry("Reorder Tags", v1beta1.NodeClassSpec{Tags: map[string]string{"keyTag-2": "valueTag-2", "keyTag-1": "valueTag-1"}}),
			Entry("Reorder BlockDeviceMapping", v1beta1.NodeClassSpec{BlockDeviceMappings: []*v1beta1.BlockDeviceMapping{{DeviceName: aws.String("map-device-2")}, {DeviceName: aws.String("map-device-1")}}}),
		)
		It("should not change hash when behavior/dynamic fields are updated", func() {
			hash := nodeClass.Hash()

			// Update a behavior/dynamic field
			nodeClass.Spec.SubnetSelectorTerms = []v1beta1.SubnetSelectorTerm{
				{
					Tags: map[string]string{"subnet-test-key": "subnet-test-value"},
				},
			}
			nodeClass.Spec.SecurityGroupSelectorTerms = []v1beta1.SecurityGroupSelectorTerm{
				{
					Tags: map[string]string{"sg-test-key": "sg-test-value"},
				},
			}
			nodeClass.Spec.AMISelectorTerms = []v1beta1.AMISelectorTerm{
				{
					Tags: map[string]string{"ami-test-key": "ami-test-value"},
				},
			}
			updatedHash := nodeClass.Hash()
			Expect(hash).To(Equal(updatedHash))
		})
		It("should expect two provisioner with the same spec to have the same provisioner hash", func() {
			otherNodeClass := test.NodeClass(v1beta1.NodeClass{
				Spec: nodeClass.Spec,
			})
			Expect(nodeClass.Hash()).To(Equal(otherNodeClass.Hash()))
		})
	})
})
