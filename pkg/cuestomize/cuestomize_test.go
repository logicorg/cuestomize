package cuestomize

import (
	"testing"

	"github.com/logicorg/cuestomize/api"
	"github.com/logicorg/cuestomize/pkg/cuestomize/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/resid"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

const pruneNullsItems = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: deploy1
  namespace: app
  labels: null
  annotations: null
spec:
  replicas: null
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: config1
  namespace: app
data:
  foo: null
`

var pruneNullsIncludes = []types.Selector{
	{ResId: resid.ResId{Gvk: resid.Gvk{Kind: "Deployment"}}},
	{ResId: resid.ResId{Gvk: resid.Gvk{Kind: "ConfigMap"}}},
}

// TestCuestomize_PruneDoesNotModifyItems checks that pruning nulls only affects the includes seen by CUE,
// while the items passed back to kustomize are left untouched.
func TestCuestomize_PruneDoesNotModifyItems(t *testing.T) {
	tests := []struct {
		name       string
		modelPath  string
		config     *api.KRMInput
		shouldFail bool
	}{
		{
			name:      "model rejects unpruned nulls",
			modelPath: "../../testdata/function/cue-modules/prune-nulls-model",
			config: &api.KRMInput{
				TypeMeta:   metav1.TypeMeta{APIVersion: "cuestomize.dev/v1alpha1", Kind: "Cuestomization"},
				ObjectMeta: metav1.ObjectMeta{Name: "prune-nulls"},
				Input:      map[string]interface{}{},
				Includes:   pruneNullsIncludes,
			},
			shouldFail: true,
		},
		{
			name:      "prune nulls",
			modelPath: "../../testdata/function/cue-modules/prune-nulls-model-replicas",
			config: &api.KRMInput{
				TypeMeta: metav1.TypeMeta{APIVersion: "cuestomize.dev/v1alpha1", Kind: "Cuestomization"},
				ObjectMeta: metav1.ObjectMeta{
					Name:        "prune-nulls",
					Annotations: map[string]string{PruneNullsAnnotationKey: "true"},
				},
				Input:    map[string]interface{}{},
				Includes: pruneNullsIncludes,
			},
		},
		{
			name:      "prune nulls in validator mode",
			modelPath: "../../testdata/function/cue-modules/prune-nulls-model-replicas",
			config: &api.KRMInput{
				TypeMeta: metav1.TypeMeta{APIVersion: "cuestomize.dev/v1alpha1", Kind: "Cuestomization"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "prune-nulls",
					Annotations: map[string]string{
						PruneNullsAnnotationKey: "true",
						ValidatorAnnotationKey:  "true",
					},
				},
				Input:    map[string]interface{}{},
				Includes: pruneNullsIncludes,
			},
		},
		{
			name:      "prune null labels and annotations",
			modelPath: "../../testdata/function/cue-modules/prune-nulls-model",
			config: &api.KRMInput{
				TypeMeta: metav1.TypeMeta{APIVersion: "cuestomize.dev/v1alpha1", Kind: "Cuestomization"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "prune-nulls",
					Annotations: map[string]string{
						PruneLabelsAnnotationKey:      "true",
						PruneAnnotationsAnnotationKey: "true",
					},
				},
				Input:    map[string]interface{}{},
				Includes: pruneNullsIncludes,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			items, err := kio.FromBytes([]byte(pruneNullsItems))
			require.NoError(t, err)

			// snapshot the items before running, so we can compare them after running Cuestomize
			snapshots := make(map[resid.ResId]string, len(items))
			for _, item := range items {
				snapshots[resid.FromRNode(item)] = item.MustString()
			}

			// the model fails to unify unless the includes' nulls are pruned
			result, err := Cuestomize(t.Context(), items, tt.config, WithModelProvider(model.NewLocalPathProvider(tt.modelPath)))
			if tt.shouldFail {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			got := make(map[resid.ResId]*kyaml.RNode, len(result))
			for _, res := range result {
				got[resid.FromRNode(res)] = res
			}
			for id, snapshot := range snapshots {
				require.Contains(t, got, id)
				assert.Equal(t, snapshot, got[id].MustString(), "item %s was modified", id)
			}
		})
	}
}
