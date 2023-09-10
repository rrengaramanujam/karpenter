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

package fake

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/aws/karpenter-core/pkg/apis/v1beta1"
	corecloudprovider "github.com/aws/karpenter-core/pkg/cloudprovider"
	"github.com/aws/karpenter-core/pkg/test"
	"github.com/aws/karpenter/pkg/apis/v1alpha1"
)

const (
	defaultRegion = "us-west-2"
)

var _ corecloudprovider.CloudProvider = (*CloudProvider)(nil)

type CloudProvider struct {
	InstanceTypes []*corecloudprovider.InstanceType
	ValidAMIs     []string
}

func (c *CloudProvider) Create(_ context.Context, _ *v1beta1.NodeClaim) (*v1beta1.NodeClaim, error) {
	name := test.RandomName()
	return &v1beta1.NodeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Status: v1beta1.NodeClaimStatus{
			ProviderID: RandomProviderID(),
		},
	}, nil
}

func (c *CloudProvider) GetInstanceTypes(_ context.Context, _ *v1beta1.NodePool) ([]*corecloudprovider.InstanceType, error) {
	if c.InstanceTypes != nil {
		return c.InstanceTypes, nil
	}
	return []*corecloudprovider.InstanceType{
		{Name: "default-instance-type"},
	}, nil
}

func (c *CloudProvider) IsDrifted(_ context.Context, nodeClaim *v1beta1.NodeClaim) (corecloudprovider.DriftReason, error) {
	nodeAMI := nodeClaim.Labels[v1alpha1.LabelInstanceAMIID]
	for _, ami := range c.ValidAMIs {
		if nodeAMI == ami {
			return "", nil
		}
	}
	return "drifted", nil
}

func (c *CloudProvider) Get(context.Context, string) (*v1beta1.NodeClaim, error) {
	return nil, nil
}

func (c *CloudProvider) List(context.Context) ([]*v1beta1.NodeClaim, error) {
	return nil, nil
}

func (c *CloudProvider) Delete(context.Context, *v1beta1.NodeClaim) error {
	return nil
}

// Name returns the CloudProvider implementation name.
func (c *CloudProvider) Name() string {
	return "fake"
}
