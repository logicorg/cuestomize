package api

import (
	"cuelang.org/go/cue"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

// Includes is a map that holds manifests, indexed by their API version, kind, namespace, and name respectively.
type Includes map[string]map[string]map[string]map[string]interface{}

// IntoCueValue tries to convert the Includes into a CUE value.
func (i Includes) IntoCueValue(cueCtx *cue.Context) (*cue.Value, error) {
	return IntoCueValue(cueCtx, i)
}

// Add adds an include to the Includes map.
func (i Includes) Add(include *kyaml.RNode) {
	i.initialiseMap(include)

	i[include.GetApiVersion()][include.GetKind()][include.GetNamespace()][include.GetName()] = any(include)
}

// PruneNullMetadata prunes labels or annotations set to explicit null if requested
func (i Includes) PruneNullMetadata(labels, annotations bool) {
	for _, o := range i { // apiVersion
		for _, o := range o { // kind
			for _, o := range o { // namespace
				for _, o := range o { // name
					cleanObject(o, labels, annotations)
				}
			}
		}
	}
}

func (i Includes) initialiseMap(include *kyaml.RNode) {
	apiVersion := include.GetApiVersion()
	kind := include.GetKind()
	name := include.GetName()
	namespace := include.GetNamespace()

	// apiVersion
	if _, ok := i[apiVersion]; !ok {
		i[apiVersion] = make(map[string]map[string]map[string]interface{})
	}
	// kind
	if _, ok := i[apiVersion][kind]; !ok {
		i[apiVersion][kind] = make(map[string]map[string]interface{})
	}
	// namespace
	if _, ok := i[apiVersion][kind][namespace]; !ok {
		i[apiVersion][kind][namespace] = make(map[string]interface{})
	}
	// name
	if _, ok := i[apiVersion][kind][namespace][name]; !ok {
		i[apiVersion][kind][namespace][name] = make(map[string]interface{})
	}
}

func dropNullLabelsAnnotations(md interface{}, labels, annotations bool) {
	if m, ok := md.(map[string]interface{}); ok {
		if labels {
			if x, exists := m["labels"]; exists && x == nil {
				delete(m, "labels")
			}
		}
		if annotations {
			if x, exists := m["annotations"]; exists && x == nil {
				delete(m, "annotations")
			}
		}
	}
}

func cleanObject(o interface{}, labels, annotations bool) {
	obj := o.(map[string]interface{})

	dropNullLabelsAnnotations(obj["metadata"], labels, annotations)

	if spec, ok := obj["spec"].(map[string]interface{}); ok {
		if tmpl, ok := spec["template"].(map[string]interface{}); ok {
			dropNullLabelsAnnotations(tmpl["metadata"], labels, annotations)
		}
	}
}
