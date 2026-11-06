package api

import (
	"slices"

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
	_ = i.forEach(func(o interface{}) error {
		pruneNullMetadata(o, labels, annotations)
		return nil
	})
}

// PruneNulls prunes all fields set to explicit null, at any depth.
// Null items in sequences are kept, as removing them would shift the other items' indexes.
func (i Includes) PruneNulls() error {
	return i.forEach(func(o interface{}) error {
		rn := o.(*kyaml.RNode)
		_, err := rn.Pipe(PruneNullsFilter(nullsPreservedFields(rn)...))
		return err
	})
}

func (i Includes) forEach(fn func(o interface{}) error) error {
	for _, o := range i { // apiVersion
		for _, o := range o { // kind
			for _, o := range o { // namespace
				for _, o := range o { // name
					if err := fn(o); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
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

func pruneNullMetadata(o interface{}, labels, annotations bool) {
	rNode := o.(*kyaml.RNode)

	for _, path := range [][]string{{"metadata"}, {"spec", "template", "metadata"}} {
		meta, err := rNode.Pipe(kyaml.Lookup(path...))
		if err != nil || meta.IsNil() {
			continue
		}
		for field, enabled := range map[string]bool{"labels": labels, "annotations": annotations} {
			if !enabled {
				continue
			}
			if v, err := meta.Pipe(kyaml.Get(field)); err == nil && v.IsTaggedNull() {
				_, _ = meta.Pipe(kyaml.Clear(field))
			}
		}
	}
}

// nullsPreserved lists, per v1 kind, the top-level string maps whose null values must be kept:
// in a string map, a null value sets the key to "" instead of leaving it unset.
var nullsPreserved = map[string][]string{
	"ConfigMap": {"data", "binaryData"},
	"Secret":    {"data", "stringData"},
}

func nullsPreservedFields(rn *kyaml.RNode) []string {
	if rn.GetApiVersion() != "v1" {
		return nil
	}
	return nullsPreserved[rn.GetKind()]
}

// PruneNullsFilter returns a filter that prunes all map fields set to explicit null, at any depth.
// Null items in sequences are kept, as removing them would shift the other items' indexes.
// The content of the top-level fields named in preserve is left untouched,
// though the fields themselves are still pruned when set to null.
func PruneNullsFilter(preserve ...string) kyaml.Filter {
	return kyaml.FilterFunc(func(rn *kyaml.RNode) (*kyaml.RNode, error) {
		return rn, pruneNulls(rn, preserve)
	})
}

func pruneNulls(rn *kyaml.RNode, preserve []string) error {
	if rn.IsNil() {
		return nil
	}

	switch rn.YNode().Kind {
	case kyaml.MappingNode:
		return rn.VisitFields(func(f *kyaml.MapNode) error {
			if f.Value.IsTaggedNull() {
				_, err := rn.Pipe(kyaml.Clear(f.Key.YNode().Value))
				return err
			}
			if slices.Contains(preserve, f.Key.YNode().Value) {
				return nil
			}
			return pruneNulls(f.Value, nil)
		})
	case kyaml.SequenceNode:
		return rn.VisitElements(func(e *kyaml.RNode) error {
			return pruneNulls(e, nil)
		})
	}
	return nil
}
