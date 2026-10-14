package cuestomize

import (
	"context"

	"cuelang.org/go/cue/build"
	"github.com/logicorg/cuestomize/api"
	"github.com/logicorg/cuestomize/pkg/cuerrors"
)

const (
	// ValidatorAnnotationKey is the annotation key that marks a CUE function as a validator.
	ValidatorAnnotationKey = "config.cuestomize.io/validator"

	// PruneLabelsAnnotationKey is the annotation key to control null labels pruning
	PruneLabelsAnnotationKey = "config.cuestomize.io/prune-null-labels"
	// PruneAnnotationsAnnotationKey is the annotation key to control null annotations pruning
	PruneAnnotationsAnnotationKey = "config.cuestomize.io/prune-null-annotations"

	trueStr = "true"
)

// CheckInstances checks if any of the instances have an error and returns an error if so.
func CheckInstances(ctx context.Context, instances []*build.Instance) error {
	detailer := cuerrors.FromContextOrEmpty(ctx)

	for _, inst := range instances {
		if inst.Err != nil {
			return detailer.ErrorWithDetails(inst.Err, "failed to load CUE instance")
		}
	}
	return nil
}

// ShouldActAsValidator checks if the KRMInput configuration has the validator annotation set.
func ShouldActAsValidator(config *api.KRMInput) bool {
	return config.Annotations != nil &&
		config.Annotations[ValidatorAnnotationKey] == trueStr
}

func ShouldPruneLabels(config *api.KRMInput) bool {
	return config.Annotations != nil &&
		config.Annotations[PruneLabelsAnnotationKey] == trueStr
}

func ShouldPruneAnnotations(config *api.KRMInput) bool {
	return config.Annotations != nil &&
		config.Annotations[PruneAnnotationsAnnotationKey] == trueStr
}
