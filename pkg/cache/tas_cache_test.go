/*
Copyright 2024 The Kubernetes Authors.

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

package cache

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kueue "sigs.k8s.io/kueue/apis/kueue/v1beta1"
	"sigs.k8s.io/kueue/pkg/resources"
	utiltesting "sigs.k8s.io/kueue/pkg/util/testing"
)

func TestFindTopologyAssignment(t *testing.T) {
	const (
		tasBlockLabel = "cloud.com/topology-block"
		tasRackLabel  = "cloud.com/topology-rack"
		tasHostLabel  = "kubernetes.io/hostname"
	)

	defaultNodes := []corev1.Node{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "b1-r1-x1",
				Labels: map[string]string{
					tasBlockLabel: "b1",
					tasRackLabel:  "r1",
					tasHostLabel:  "x1",
				},
			},
			Status: corev1.NodeStatus{
				Allocatable: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("1"),
					corev1.ResourceMemory: resource.MustParse("1Gi"),
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "b1-r2-x2",
				Labels: map[string]string{
					tasBlockLabel: "b1",
					tasRackLabel:  "r2",
					tasHostLabel:  "x1",
				},
			},
			Status: corev1.NodeStatus{
				Allocatable: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("1"),
					corev1.ResourceMemory: resource.MustParse("1Gi"),
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "b1-r2-x3",
				Labels: map[string]string{
					tasBlockLabel: "b1",
					tasRackLabel:  "r2",
					tasHostLabel:  "x3",
				},
			},
			Status: corev1.NodeStatus{
				Allocatable: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("1"),
					corev1.ResourceMemory: resource.MustParse("1Gi"),
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "b1-r2-x4",
				Labels: map[string]string{
					tasBlockLabel: "b1",
					tasRackLabel:  "r2",
					tasHostLabel:  "x4",
				},
			},
			Status: corev1.NodeStatus{
				Allocatable: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("1"),
					corev1.ResourceMemory: resource.MustParse("1Gi"),
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "b2-r1-x5",
				Labels: map[string]string{
					tasBlockLabel: "b2",
					tasRackLabel:  "r1",
					tasHostLabel:  "x5",
				},
			},
			Status: corev1.NodeStatus{
				Allocatable: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("1"),
					corev1.ResourceMemory: resource.MustParse("1Gi"),
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "b2-r2-x6",
				Labels: map[string]string{
					tasBlockLabel: "b2",
					tasRackLabel:  "r2",
					tasHostLabel:  "x6",
				},
			},
			Status: corev1.NodeStatus{
				Allocatable: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("2"),
					corev1.ResourceMemory: resource.MustParse("4Gi"),
				},
			},
		},
	}
	defaultOneLevel := []string{
		tasHostLabel,
	}
	defaultTwoLevels := []string{
		tasBlockLabel,
		tasRackLabel,
	}
	defaultThreeLevels := []string{
		tasBlockLabel,
		tasRackLabel,
		tasHostLabel,
	}

	cases := map[string]struct {
		request        kueue.PodSetTopologyRequest
		levels         []string
		nodeLabels     map[string]string
		nodes          []corev1.Node
		requests       resources.Requests
		count          int32
		wantAssignment *kueue.TopologyAssignment
	}{
		"minimize the number of used racks before optimizing the number of nodes": {
			// Solution by optimizing the number of racks then nodes: [r3]: [x3,x4,x5,x6]
			// Solution by optimizing the number of nodes: [r1,r2]: [x1,x2]
			//
			//       b1
			//   /   |    \
			//  r1   r2   r3
			//  |     |    |   \   \     \
			// x1:2,x2:2,x3:1,x4:1,x5:1,x6:1
			//
			nodes: []corev1.Node{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "b1-r1-x1",
						Labels: map[string]string{
							tasBlockLabel: "b1",
							tasRackLabel:  "r1",
							tasHostLabel:  "x1",
						},
					},
					Status: corev1.NodeStatus{
						Allocatable: corev1.ResourceList{
							corev1.ResourceCPU: resource.MustParse("2"),
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "b1-r2-x2",
						Labels: map[string]string{
							tasBlockLabel: "b1",
							tasRackLabel:  "r2",
							tasHostLabel:  "x2",
						},
					},
					Status: corev1.NodeStatus{
						Allocatable: corev1.ResourceList{
							corev1.ResourceCPU: resource.MustParse("2"),
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "b1-r3-x3",
						Labels: map[string]string{
							tasBlockLabel: "b1",
							tasRackLabel:  "r3",
							tasHostLabel:  "x3",
						},
					},
					Status: corev1.NodeStatus{
						Allocatable: corev1.ResourceList{
							corev1.ResourceCPU: resource.MustParse("1"),
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "b1-r3-x4",
						Labels: map[string]string{
							tasBlockLabel: "b1",
							tasRackLabel:  "r3",
							tasHostLabel:  "x4",
						},
					},
					Status: corev1.NodeStatus{
						Allocatable: corev1.ResourceList{
							corev1.ResourceCPU: resource.MustParse("1"),
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "b1-r3-x5",
						Labels: map[string]string{
							tasBlockLabel: "b1",
							tasRackLabel:  "r3",
							tasHostLabel:  "x5",
						},
					},
					Status: corev1.NodeStatus{
						Allocatable: corev1.ResourceList{
							corev1.ResourceCPU: resource.MustParse("1"),
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "b1-r3-x6",
						Labels: map[string]string{
							tasBlockLabel: "b1",
							tasRackLabel:  "r3",
							tasHostLabel:  "x6",
						},
					},
					Status: corev1.NodeStatus{
						Allocatable: corev1.ResourceList{
							corev1.ResourceCPU: resource.MustParse("1"),
						},
					},
				},
			},
			request: kueue.PodSetTopologyRequest{
				Required: ptr.To(tasBlockLabel),
			},
			levels: defaultThreeLevels,
			requests: resources.Requests{
				corev1.ResourceCPU: 1000,
			},
			count: 4,
			wantAssignment: &kueue.TopologyAssignment{
				Levels: defaultThreeLevels,
				Domains: []kueue.TopologyDomainAssignment{
					{
						Count: 1,
						Values: []string{
							"b1",
							"r3",
							"x3",
						},
					},
					{
						Count: 1,
						Values: []string{
							"b1",
							"r3",
							"x4",
						},
					},
					{
						Count: 1,
						Values: []string{
							"b1",
							"r3",
							"x5",
						},
					},
					{
						Count: 1,
						Values: []string{
							"b1",
							"r3",
							"x6",
						},
					},
				},
			},
		},
		"host required; single Pod fits in the host": {
			nodes: defaultNodes,
			request: kueue.PodSetTopologyRequest{
				Required: ptr.To(tasHostLabel),
			},
			levels: defaultThreeLevels,
			requests: resources.Requests{
				corev1.ResourceCPU: 1000,
			},
			count: 1,
			wantAssignment: &kueue.TopologyAssignment{
				Levels: defaultThreeLevels,
				Domains: []kueue.TopologyDomainAssignment{
					{
						Count: 1,
						Values: []string{
							"b2",
							"r2",
							"x6",
						},
					},
				},
			},
		},
		"rack required; single Pod fits in a rack": {
			nodes: defaultNodes,
			request: kueue.PodSetTopologyRequest{
				Required: ptr.To(tasRackLabel),
			},
			levels: defaultTwoLevels,
			requests: resources.Requests{
				corev1.ResourceCPU: 1000,
			},
			count: 1,
			wantAssignment: &kueue.TopologyAssignment{
				Levels: defaultTwoLevels,
				Domains: []kueue.TopologyDomainAssignment{
					{
						Count: 1,
						Values: []string{
							"b1",
							"r2",
						},
					},
				},
			},
		},
		"rack required; multiple Pods fits in a rack": {
			nodes: defaultNodes,
			request: kueue.PodSetTopologyRequest{
				Required: ptr.To(tasRackLabel),
			},
			levels: defaultTwoLevels,
			requests: resources.Requests{
				corev1.ResourceCPU: 1000,
			},
			count: 3,
			wantAssignment: &kueue.TopologyAssignment{
				Levels: defaultTwoLevels,
				Domains: []kueue.TopologyDomainAssignment{
					{
						Count: 3,
						Values: []string{
							"b1",
							"r2",
						},
					},
				},
			},
		},
		"rack required; too many pods to fit in any rack": {
			nodes: defaultNodes,
			request: kueue.PodSetTopologyRequest{
				Required: ptr.To(tasRackLabel),
			},
			levels: defaultTwoLevels,
			requests: resources.Requests{
				corev1.ResourceCPU: 1000,
			},
			count:          4,
			wantAssignment: nil,
		},
		"block required; single Pod fits in a block": {
			nodes: defaultNodes,
			request: kueue.PodSetTopologyRequest{
				Required: ptr.To(tasBlockLabel),
			},
			levels: defaultTwoLevels,
			requests: resources.Requests{
				corev1.ResourceCPU: 1000,
			},
			count: 1,
			wantAssignment: &kueue.TopologyAssignment{
				Levels: []string{
					tasBlockLabel,
					tasRackLabel,
				},
				Domains: []kueue.TopologyDomainAssignment{
					{
						Count: 1,
						Values: []string{
							"b1",
							"r2",
						},
					},
				},
			},
		},
		"block required; Pods fit in a block spread across two racks": {
			nodes: defaultNodes,
			request: kueue.PodSetTopologyRequest{
				Required: ptr.To(tasBlockLabel),
			},
			levels: defaultTwoLevels,
			requests: resources.Requests{
				corev1.ResourceCPU: 1000,
			},
			count: 4,
			wantAssignment: &kueue.TopologyAssignment{
				Levels: defaultTwoLevels,
				Domains: []kueue.TopologyDomainAssignment{
					{
						Count: 3,
						Values: []string{
							"b1",
							"r2",
						},
					},
					{
						Count: 1,
						Values: []string{
							"b1",
							"r1",
						},
					},
				},
			},
		},
		"block required; single Pod which cannot be split": {
			nodes: defaultNodes,
			request: kueue.PodSetTopologyRequest{
				Required: ptr.To(tasBlockLabel),
			},
			levels: defaultTwoLevels,
			requests: resources.Requests{
				corev1.ResourceCPU: 4000,
			},
			count:          1,
			wantAssignment: nil,
		},
		"block required; too many Pods to fit requested": {
			nodes: defaultNodes,
			request: kueue.PodSetTopologyRequest{
				Required: ptr.To(tasBlockLabel),
			},
			levels: defaultTwoLevels,
			requests: resources.Requests{
				corev1.ResourceCPU: 1000,
			},
			count:          5,
			wantAssignment: nil,
		},
		"rack required; single Pod requiring memory": {
			nodes: defaultNodes,
			request: kueue.PodSetTopologyRequest{
				Required: ptr.To(tasRackLabel),
			},
			levels: defaultTwoLevels,
			requests: resources.Requests{
				corev1.ResourceMemory: 1024,
			},
			count: 4,
			wantAssignment: &kueue.TopologyAssignment{
				Levels: defaultTwoLevels,
				Domains: []kueue.TopologyDomainAssignment{
					{
						Count: 4,
						Values: []string{
							"b2",
							"r2",
						},
					},
				},
			},
		},
		"rack preferred; but only block can accommodate the workload": {
			nodes: defaultNodes,
			request: kueue.PodSetTopologyRequest{
				Preferred: ptr.To(tasRackLabel),
			},
			levels: defaultTwoLevels,
			requests: resources.Requests{
				corev1.ResourceCPU: 1000,
			},
			count: 4,
			wantAssignment: &kueue.TopologyAssignment{
				Levels: defaultTwoLevels,
				Domains: []kueue.TopologyDomainAssignment{
					{
						Count: 3,
						Values: []string{
							"b1",
							"r2",
						},
					},
					{
						Count: 1,
						Values: []string{
							"b1",
							"r1",
						},
					},
				},
			},
		},
		"rack preferred; but only multiple blocks can accommodate the workload": {
			nodes: defaultNodes,
			request: kueue.PodSetTopologyRequest{
				Preferred: ptr.To(tasRackLabel),
			},
			levels: defaultTwoLevels,
			requests: resources.Requests{
				corev1.ResourceCPU: 1000,
			},
			count: 6,
			wantAssignment: &kueue.TopologyAssignment{
				Levels: defaultTwoLevels,
				Domains: []kueue.TopologyDomainAssignment{
					{
						Count: 3,
						Values: []string{
							"b1",
							"r2",
						},
					},
					{
						Count: 2,
						Values: []string{
							"b2",
							"r2",
						},
					},
					{
						Count: 1,
						Values: []string{
							"b1",
							"r1",
						},
					},
				},
			},
		},
		"block preferred; but only multiple blocks can accommodate the workload": {
			nodes: defaultNodes,
			request: kueue.PodSetTopologyRequest{
				Preferred: ptr.To(tasBlockLabel),
			},
			levels: defaultTwoLevels,
			requests: resources.Requests{
				corev1.ResourceCPU: 1000,
			},
			count: 6,
			wantAssignment: &kueue.TopologyAssignment{
				Levels: defaultTwoLevels,
				Domains: []kueue.TopologyDomainAssignment{
					{
						Count: 3,
						Values: []string{
							"b1",
							"r2",
						},
					},
					{
						Count: 2,
						Values: []string{
							"b2",
							"r2",
						},
					},
					{
						Count: 1,
						Values: []string{
							"b1",
							"r1",
						},
					},
				},
			},
		},
		"block preferred; but the workload cannot be accommodate in entire topology": {
			nodes: defaultNodes,
			request: kueue.PodSetTopologyRequest{
				Preferred: ptr.To(tasBlockLabel),
			},
			levels: defaultTwoLevels,
			requests: resources.Requests{
				corev1.ResourceCPU: 1000,
			},
			count:          10,
			wantAssignment: nil,
		},
		"only nodes with matching labels are considered; no matching node": {
			nodes: []corev1.Node{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "b1-r1-x1",
						Labels: map[string]string{
							"zone":       "zone-a",
							tasHostLabel: "x1",
						},
					},
					Status: corev1.NodeStatus{
						Allocatable: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("1"),
							corev1.ResourceMemory: resource.MustParse("1Gi"),
						},
					},
				},
			},
			request: kueue.PodSetTopologyRequest{
				Required: ptr.To(tasHostLabel),
			},
			nodeLabels: map[string]string{
				"zone": "zone-b",
			},
			levels: defaultOneLevel,
			requests: resources.Requests{
				corev1.ResourceCPU: 1000,
			},
			count:          1,
			wantAssignment: nil,
		},
		"only nodes with matching labels are considered; matching node is found": {
			nodes: []corev1.Node{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "b1-r1-x1",
						Labels: map[string]string{
							"zone":       "zone-a",
							tasHostLabel: "x1",
						},
					},
					Status: corev1.NodeStatus{
						Allocatable: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("1"),
							corev1.ResourceMemory: resource.MustParse("1Gi"),
						},
					},
				},
			},
			request: kueue.PodSetTopologyRequest{
				Required: ptr.To(tasHostLabel),
			},
			nodeLabels: map[string]string{
				"zone": "zone-a",
			},
			levels: defaultOneLevel,
			requests: resources.Requests{
				corev1.ResourceCPU: 1000,
			},
			count: 1,
			wantAssignment: &kueue.TopologyAssignment{
				Levels: defaultOneLevel,
				Domains: []kueue.TopologyDomainAssignment{
					{
						Count: 1,
						Values: []string{
							"x1",
						},
					},
				},
			},
		},
		"only nodes with matching levels are considered; no host label on node": {
			nodes: []corev1.Node{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "b1-r1-x1",
						Labels: map[string]string{
							tasBlockLabel: "b1",
							tasRackLabel:  "r1",
							// the node doesn't have the tasHostLabel required by topology
						},
					},
					Status: corev1.NodeStatus{
						Allocatable: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("1"),
							corev1.ResourceMemory: resource.MustParse("1Gi"),
						},
					},
				},
			},
			request: kueue.PodSetTopologyRequest{
				Required: ptr.To(tasRackLabel),
			},
			levels: defaultThreeLevels,
			requests: resources.Requests{
				corev1.ResourceCPU: 1000,
			},
			count:          1,
			wantAssignment: nil,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()

			initialObjects := make([]client.Object, 0)
			for i := range tc.nodes {
				initialObjects = append(initialObjects, &tc.nodes[i])
			}
			client := utiltesting.NewFakeClient(initialObjects...)
			tasCache := NewTASCache(client)
			tasFlavorCache := tasCache.NewTASFlavorCache(tc.levels, tc.nodeLabels)
			snapshot := tasFlavorCache.snapshot(ctx)
			gotAssignment := snapshot.FindTopologyAssignment(&tc.request, tc.requests, tc.count)
			if diff := cmp.Diff(tc.wantAssignment, gotAssignment); diff != "" {
				t.Errorf("unexpected topology assignment (-want,+got): %s", diff)
			}
		})
	}
}
