/*
Copyright 2026 Serge Logvinov.

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

package formatter_test

import (
	"testing"

	"github.com/sergelogvinov/mimiops-mcp/internal/tools"
	"github.com/sergelogvinov/mimiops-mcp/pkg/formatter"
	"github.com/stretchr/testify/assert"
)

// Local structs exercising nested-struct headings and map[string]any values,
// which the tools types only cover via embedded or sliced structs.
type deeperBlock struct {
	Detail string `json:"detail" jsonschema:"Detail of the deeper block"`
}

type nestedBlock struct {
	Deeper deeperBlock `json:"deeper" jsonschema:"Deeper block"`
}

type holder struct {
	Name    string      `json:"name" jsonschema:"Name"`
	Nested  nestedBlock `json:"nested" jsonschema:"Nested block"`
	Skipped string      `json:"skipped"`
}

type details struct {
	Spec map[string]any `json:"spec,omitempty" jsonschema:"Spec of the workload"`
}

type EmbeddedDetail struct {
	Detail string `json:"detail" jsonschema:"Detail"`
}

type TableRow struct {
	*EmbeddedDetail

	Name string `json:"name" jsonschema:"Name"`
}

type RowsHolder struct {
	Rows []TableRow `json:"rows" jsonschema:"Rows"`
}

type PtrKeyMap struct {
	Values map[*string]string `json:"values,omitempty" jsonschema:"Values"`
}

type PtrPtrRow struct {
	Name string `json:"name" jsonschema:"Name"`
}

type PtrPtrRows struct {
	Rows []**PtrPtrRow `json:"rows,omitempty" jsonschema:"Rows"`
}

func TestToText(t *testing.T) {
	for _, tt := range []struct {
		name     string
		input    any
		expected string
	}{
		{
			name: "pod summary with owner references table",
			input: tools.PodSummary{
				Namespace: "default",
				Name:      "my-pod",
				Ready:     "1/1",
				Restarts:  3,
				Age:       "5d",
				Status:    "Running",
				Node:      "node-1",
				OwnerReferences: []tools.OwnerReference{
					{APIVersion: "apps/v1", Kind: "ReplicaSet", Name: "web-abc123"},
				},
			},
			expected: `Namespace: default
Name: my-pod
Ready status (e.g., 1/2): 1/1
Restarts: 3
Age: 5d
Status: Running
Node: node-1

### Owner References

| apiVersion | kind | name |
| --- | --- | --- |
| apps/v1 | ReplicaSet | web-abc123 |`,
		},
		{
			name: "zero values and empty slice are omitted or printed per omitempty",
			input: tools.PodSummary{
				Name: "my-pod",
			},
			expected: `Name: my-pod
Ready status (e.g., 1/2):
Restarts: 0
Age:
Status:
Node:`,
		},
		{
			name: "node summary flattens embedded capacity struct",
			input: tools.NodeSummary{
				NodeCapacityInfo: tools.NodeCapacityInfo{
					CPU:    "4",
					Memory: "8Gi",
					Pods:   110,
				},
				Name:           "node-1",
				Status:         "Ready",
				Roles:          []string{"control-plane", "worker"},
				Age:            "10d",
				KubeletVersion: "v1.31.1",
				ImageVersion:   "debian12",
				InternalIPs:    "10.0.0.1",
			},
			expected: `CPU capacity of the node: 4
Memory capacity of the node: 8Gi
Maximum number of pods the node can run: 110
Name of the node: node-1
Status of the node: Ready
Roles of the node: control-plane, worker
Age of the node: 10d
Kubelet version of the node: v1.31.1
OS image version of the node: debian12
Internal IP addresses of the node: 10.0.0.1`,
		},
		{
			name: "node spec renders taints table with omitted empty cell",
			input: tools.NodeSpec{
				Unschedulable: true,
				Taints: []tools.TaintInfo{
					{Key: "node-role.kubernetes.io/control-plane", Effect: "NoSchedule"},
				},
			},
			expected: `Whether the node is unschedulable: true

### List of taints

| Key | Value | Effect |
| --- | --- | --- |
| node-role.kubernetes.io/control-plane |  | NoSchedule |`,
		},
		{
			name: "pod spec renders containers table and sorted node selector",
			input: tools.PodSpec{
				RestartPolicy: "Always",
				Containers: []tools.ContainerInfo{
					{
						Name:     "web",
						Image:    "nginx:1.27",
						Ports:    []string{"8080/tcp", "9090/tcp"},
						Requests: map[string]string{"cpu": "100m", "memory": "128Mi"},
						Limits:   map[string]string{"cpu": "500m"},
					},
				},
				NodeSelector: map[string]string{"pool": "default", "zone": "b"},
			},
			expected: `restart policy: Always
Node selector: pool=default, zone=b

### List of containers

| Name | Image | List of ports | Resource requests | Resource limits |
| --- | --- | --- | --- | --- |
| web | nginx:1.27 | 8080/tcp, 9090/tcp | cpu=100m, memory=128Mi | cpu=500m |`,
		},
		{
			name: "limit range spec renders map cells",
			input: tools.LimitRangeSpec{
				Limits: []tools.LimitRangeLimit{
					{
						Type:    "Container",
						Min:     map[string]string{"cpu": "100m"},
						Default: map[string]string{"cpu": "500m", "memory": "256Mi"},
					},
				},
			},
			expected: "### List of limits\n\n" +
				"| Type of the limit (Container, Pod, or PersistentVolumeClaim) | Min resource constraints | " +
				"Max resource constraints | Default resource constraints | Default request resource constraints | " +
				"Max limit to request ratio |\n" +
				"| --- | --- | --- | --- | --- | --- |\n" +
				"| Container | cpu=100m |  | cpu=500m, memory=256Mi |  |  |",
		},
		{
			name: "nested structs become deeper headings",
			input: holder{
				Name:   "root",
				Nested: nestedBlock{Deeper: deeperBlock{Detail: "value"}},
			},
			expected: `Name: root

### Nested block

#### Deeper block

Detail of the deeper block: value`,
		},
		{
			name: "fields without jsonschema tag are skipped",
			input: holder{
				Name:    "root",
				Skipped: "invisible",
			},
			expected: `Name: root

### Nested block

#### Deeper block

Detail of the deeper block:`,
		},
		{
			name: "map of arbitrary values is sorted and stringified",
			input: details{
				Spec: map[string]any{"replicas": 3, "name": "web", "paused": true, "ratio": 1.5},
			},
			expected: `Spec of the workload: name=web, paused=true, ratio=1.5, replicas=3`,
		},
		{
			name: "pvc summary prints zero values without omitempty",
			input: tools.PVCSummary{
				Name:      "data",
				Namespace: "default",
			},
			expected: `Name of the PVC: data
Namespace of the PVC: default
Status of the PVC (Pending, Bound, or Lost):
Age of the PVC:`,
		},
		{
			name: "table rows with nil embedded struct pointer render an empty cell",
			input: RowsHolder{
				Rows: []TableRow{
					{EmbeddedDetail: &EmbeddedDetail{Detail: "x"}, Name: "a"},
					{Name: "b"}, // nil embedded pointer must not panic
				},
			},
			expected: `### Rows

| Detail | Name |
| --- | --- |
| x | a |
|  | b |`,
		},
		{
			name: "map with a nil pointer key renders the nil marker",
			input: func() any {
				key := "key1"

				return PtrKeyMap{
					Values: map[*string]string{nil: "none", &key: "some"},
				}
			}(),
			expected: `Values: <nil>=none, key1=some`,
		},
		{
			name: "slice of multiply pointer-indirected rows renders without panic",
			input: func() any {
				row := &PtrPtrRow{Name: "a"}
				double := &row
				nilInner := (*PtrPtrRow)(nil)

				return PtrPtrRows{
					Rows: []**PtrPtrRow{double, &nilInner, nil},
				}
			}(),
			expected: `### Rows

| Name |
| --- |
| a |`,
		},
		{
			name:     "non-struct input",
			input:    42,
			expected: "",
		},
		{
			name:     "nil input",
			input:    nil,
			expected: "",
		},
		{
			name:     "nil pointer",
			input:    (*tools.PodSummary)(nil),
			expected: "",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, formatter.ToText(tt.input))
		})
	}
}

func TestFallbackTextPointer(t *testing.T) {
	pod := &tools.PodSummary{
		Namespace: "default",
		Name:      "my-pod",
		Status:    "Running",
	}

	assert.Equal(t, formatter.ToText(*pod), formatter.ToText(pod))
}
