package cuestomize

import (
	"testing"

	"github.com/logicorg/cuestomize/api"
	"github.com/stretchr/testify/assert"
)

func TestShouldPruneNulls(t *testing.T) {
	tests := []struct {
		name        string
		annotations map[string]string
		expected    bool
	}{
		{name: "nil annotations", annotations: nil, expected: false},
		{name: "annotation missing", annotations: map[string]string{}, expected: false},
		{name: "annotation true", annotations: map[string]string{PruneNullsAnnotationKey: "true"}, expected: true},
		{name: "annotation false", annotations: map[string]string{PruneNullsAnnotationKey: "false"}, expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			config := &api.KRMInput{}
			config.Annotations = tt.annotations
			assert.Equal(t, tt.expected, ShouldPruneNulls(config))
		})
	}
}
