package oci

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNames(t *testing.T) {
	r, err := ParseRepositoryName("docker-remote/ubuntu")
	assert.NoError(t, err)
	r = r.WithNamespace("library")
	assert.Equal(t, "library/ubuntu", r.String())
	assert.Equal(t, "library", r.Head())
	assert.Equal(t, "ubuntu", r.Rest())
	assert.Equal(t, []string([]string{"library", "ubuntu"}), r.Components())
	assert.Equal(t, false, r.IsSingleComponentRest())
	assert.Equal(t, "library/ubuntu", r.WithNamespace("library").String())

}
