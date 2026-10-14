package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestIncludes_Add(t *testing.T) {
	tests := []struct {
		name     string
		nodes    []*kyaml.RNode
		validate func(t *testing.T, includes Includes)
	}{
		{
			name: "add single item",
			nodes: []*kyaml.RNode{
				createTestNode(t, "v1", "ConfigMap", "default", "config1"),
			},
			validate: func(t *testing.T, includes Includes) {
				assert.Len(t, includes, 1)
				assert.Contains(t, includes, "v1")
				assert.Contains(t, includes["v1"], "ConfigMap")
				assert.Contains(t, includes["v1"]["ConfigMap"], "default")
				assert.Contains(t, includes["v1"]["ConfigMap"]["default"], "config1")
				assert.NotNil(t, includes["v1"]["ConfigMap"]["default"]["config1"])
			},
		},
		{
			name: "add multiple items with different apiVersions",
			nodes: []*kyaml.RNode{
				createTestNode(t, "v1", "ConfigMap", "default", "config1"),
				createTestNode(t, "apps/v1", "Deployment", "app", "deploy1"),
			},
			validate: func(t *testing.T, includes Includes) {
				assert.Len(t, includes, 2) // v1 and apps/v1

				// Check v1/ConfigMap/default/config1
				assert.Contains(t, includes, "v1")
				assert.Contains(t, includes["v1"], "ConfigMap")
				assert.Contains(t, includes["v1"]["ConfigMap"]["default"], "config1")

				// Check apps/v1/Deployment/app/deploy1
				assert.Contains(t, includes, "apps/v1")
				assert.Contains(t, includes["apps/v1"], "Deployment")
				assert.Contains(t, includes["apps/v1"]["Deployment"]["app"], "deploy1")
			},
		},
		{
			name: "add multiple items with different properties",
			nodes: []*kyaml.RNode{
				createTestNode(t, "v1", "ConfigMap", "default", "config1"),
				createTestNode(t, "v1", "Secret", "default", "secret1"),
				createTestNode(t, "apps/v1", "Deployment", "app", "deploy1"),
			},
			validate: func(t *testing.T, includes Includes) {
				assert.Len(t, includes, 2) // v1 and apps/v1

				// Check v1/ConfigMap/default/config1
				assert.Contains(t, includes, "v1")
				assert.Contains(t, includes["v1"], "ConfigMap")
				assert.Contains(t, includes["v1"]["ConfigMap"]["default"], "config1")

				// Check v1/Secret/default/secret1
				assert.Contains(t, includes["v1"], "Secret")
				assert.Contains(t, includes["v1"]["Secret"]["default"], "secret1")

				// Check apps/v1/Deployment/app/deploy1
				assert.Contains(t, includes, "apps/v1")
				assert.Contains(t, includes["apps/v1"], "Deployment")
				assert.Contains(t, includes["apps/v1"]["Deployment"]["app"], "deploy1")
			},
		},
		{
			name: "add items with same apiVersion and kind but different namespace",
			nodes: []*kyaml.RNode{
				createTestNode(t, "v1", "ConfigMap", "default", "config1"),
				createTestNode(t, "v1", "ConfigMap", "kube-system", "config2"),
			},
			validate: func(t *testing.T, includes Includes) {
				assert.Len(t, includes["v1"]["ConfigMap"], 2)
				assert.Contains(t, includes["v1"]["ConfigMap"]["default"], "config1")
				assert.Contains(t, includes["v1"]["ConfigMap"]["kube-system"], "config2")
			},
		},
		{
			name: "add items with same apiVersion, kind, namespace but different names",
			nodes: []*kyaml.RNode{
				createTestNode(t, "v1", "ConfigMap", "default", "config1"),
				createTestNode(t, "v1", "ConfigMap", "default", "config2"),
			},
			validate: func(t *testing.T, includes Includes) {
				assert.Len(t, includes["v1"]["ConfigMap"]["default"], 2)
				assert.Contains(t, includes["v1"]["ConfigMap"]["default"], "config1")
				assert.Contains(t, includes["v1"]["ConfigMap"]["default"], "config2")
			},
		},
		{
			name: "add items in reverse order",
			nodes: []*kyaml.RNode{
				createTestNode(t, "apps/v1", "Deployment", "app", "deploy1"),
				createTestNode(t, "v1", "Secret", "default", "secret1"),
				createTestNode(t, "v1", "ConfigMap", "default", "config1"),
			},
			validate: func(t *testing.T, includes Includes) {
				assert.Len(t, includes, 2) // v1 and apps/v1

				// Verify all items exist
				assert.Contains(t, includes["v1"]["ConfigMap"]["default"], "config1")
				assert.Contains(t, includes["v1"]["Secret"]["default"], "secret1")
				assert.Contains(t, includes["apps/v1"]["Deployment"]["app"], "deploy1")
			},
		},
		{
			name: "add items with empty namespace",
			nodes: []*kyaml.RNode{
				createTestNode(t, "v1", "ConfigMap", "", "config1"),
			},
			validate: func(t *testing.T, includes Includes) {
				assert.Contains(t, includes["v1"]["ConfigMap"][""], "config1")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			includes := make(Includes)

			for _, node := range tt.nodes {
				includes.Add(node)
			}

			if tt.validate != nil {
				tt.validate(t, includes)
			}
		})
	}
}

func createTestNode(t *testing.T, apiVersion, kind, namespace, name string) *kyaml.RNode {
	t.Helper()

	yamlObj := map[string]interface{}{}
	yamlObj["apiVersion"] = apiVersion
	yamlObj["kind"] = kind
	yamlObj["metadata"] = map[string]interface{}{
		"name":      name,
		"namespace": namespace,
		"spec": map[string]interface{}{
			"replicas": 3,
		},
	}

	node, err := kyaml.FromMap(yamlObj)
	require.NoError(t, err)
	return node
}

func TestIncludes_PruneNullMetadata(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		labels      bool
		annotations bool
		expected    string
	}{
		{
			name: "prune null labels and annotations",
			input: `apiVersion: v1
kind: ConfigMap
metadata:
  name: config1
  labels: null
  annotations: null
`,
			labels:      true,
			annotations: true,
			expected: `apiVersion: v1
kind: ConfigMap
metadata:
  name: config1
`,
		},
		{
			name: "prune only null labels",
			input: `apiVersion: v1
kind: ConfigMap
metadata:
  name: config1
  labels: null
  annotations: null
`,
			labels: true,
			expected: `apiVersion: v1
kind: ConfigMap
metadata:
  name: config1
  annotations: null
`,
		},
		{
			name: "prune only null annotations",
			input: `apiVersion: v1
kind: ConfigMap
metadata:
  name: config1
  labels: null
  annotations: null
`,
			annotations: true,
			expected: `apiVersion: v1
kind: ConfigMap
metadata:
  name: config1
  labels: null
`,
		},
		{
			name: "prune nothing when disabled",
			input: `apiVersion: v1
kind: ConfigMap
metadata:
  name: config1
  labels: null
  annotations: null
`,
			expected: `apiVersion: v1
kind: ConfigMap
metadata:
  name: config1
  labels: null
  annotations: null
`,
		},
		{
			name: "prune tilde and empty null values",
			input: `apiVersion: v1
kind: ConfigMap
metadata:
  name: config1
  labels: ~
  annotations:
`,
			labels:      true,
			annotations: true,
			expected: `apiVersion: v1
kind: ConfigMap
metadata:
  name: config1
`,
		},
		{
			name: "keep empty and non-empty maps",
			input: `apiVersion: v1
kind: ConfigMap
metadata:
  name: config1
  labels: {}
  annotations:
    foo: bar
`,
			labels:      true,
			annotations: true,
			expected: `apiVersion: v1
kind: ConfigMap
metadata:
  name: config1
  labels: {}
  annotations:
    foo: bar
`,
		},
		{
			name: "no labels or annotations",
			input: `apiVersion: v1
kind: ConfigMap
metadata:
  name: config1
`,
			labels:      true,
			annotations: true,
			expected: `apiVersion: v1
kind: ConfigMap
metadata:
  name: config1
`,
		},
		{
			name: "prune pod template metadata",
			input: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: deploy1
  namespace: app
  labels: null
spec:
  template:
    metadata:
      labels: null
      annotations: null
`,
			labels:      true,
			annotations: true,
			expected: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: deploy1
  namespace: app
spec:
  template:
    metadata: {}
`,
		},
		{
			name: "keep null outside metadata",
			input: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: deploy1
spec:
  labels: null
  template:
    spec:
      annotations: null
`,
			labels:      true,
			annotations: true,
			expected: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: deploy1
spec:
  labels: null
  template:
    spec:
      annotations: null
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			node, err := kyaml.Parse(tt.input)
			require.NoError(t, err)

			includes := make(Includes)
			includes.Add(node)
			includes.PruneNullMetadata(tt.labels, tt.annotations)

			got := includes[node.GetApiVersion()][node.GetKind()][node.GetNamespace()][node.GetName()].(*kyaml.RNode)
			assert.Equal(t, tt.expected, got.MustString())
		})
	}
}

func TestIncludes_PruneNullMetadata_MultipleIncludes(t *testing.T) {
	inputs := []string{
		"apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: config1\n  namespace: default\n  labels: null\n",
		"apiVersion: v1\nkind: Secret\nmetadata:\n  name: secret1\n  namespace: kube-system\n  annotations: null\n",
		"apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: deploy1\n  namespace: app\n  labels: null\n  annotations: null\n",
	}

	includes := make(Includes)
	var nodes []*kyaml.RNode
	for _, in := range inputs {
		node, err := kyaml.Parse(in)
		require.NoError(t, err)
		includes.Add(node)
		nodes = append(nodes, node)
	}

	includes.PruneNullMetadata(true, true)

	for _, node := range nodes {
		meta, err := node.Pipe(kyaml.Lookup("metadata"))
		require.NoError(t, err)
		assert.Nil(t, meta.Field("labels"), "labels not pruned for %s", node.GetName())
		assert.Nil(t, meta.Field("annotations"), "annotations not pruned for %s", node.GetName())
	}
}
