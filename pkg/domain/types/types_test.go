package types_test

import (
	"testing"

	"github.com/m-mizutani/zenv/pkg/domain/types"
	"github.com/stretchr/testify/assert"
)

func TestKeychainNamespace(t *testing.T) {
	t.Run("pass valid namespace", func(t *testing.T) {
		assert.NoError(t, types.NamespaceSuffix("@blue").Validate())
		assert.NoError(t, types.NamespaceSuffix("@@orange").Validate()) // Not prohibit double @
	})

	t.Run("fail invalid namespace", func(t *testing.T) {
		assert.ErrorIs(t, types.NamespaceSuffix("blue").Validate(), types.ErrInvalidArgument)
		assert.ErrorIs(t, types.NamespaceSuffix("@").Validate(), types.ErrInvalidArgument)
	})

	t.Run("bind prefix and namespace", func(t *testing.T) {
		assert.Equal(t, types.Namespace("magic.blue"), types.NamespaceSuffix("@blue").ToNamespace(types.NamespacePrefix("magic.")))
	})
}
