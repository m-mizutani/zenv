package types_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/zenv/pkg/domain/types"
)

func TestKeychainNamespace(t *testing.T) {
	t.Run("pass valid namespace", func(t *testing.T) {
		gt.NoError(t, types.NamespaceSuffix("@blue").Validate())
		gt.NoError(t, types.NamespaceSuffix("@@orange").Validate()) // Not prohibit double @
	})

	t.Run("fail invalid namespace", func(t *testing.T) {
		gt.Error(t, types.NamespaceSuffix("blue").Validate()).
			Is(types.ErrInvalidArgument)
		gt.Error(t, types.NamespaceSuffix("@").Validate()).
			Is(types.ErrInvalidArgument)
	})

	t.Run("bind prefix and namespace", func(t *testing.T) {
		gt.Value(t, types.Namespace("magic.blue")).
			Equal(types.NamespaceSuffix("@blue").ToNamespace(types.NamespacePrefix("magic.")))
	})
}
