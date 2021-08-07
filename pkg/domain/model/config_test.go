package model_test

import (
	"testing"

	"github.com/m-mizutani/zenv/pkg/domain/model"
	"github.com/stretchr/testify/assert"
)

func TestKeychainNamespace(t *testing.T) {
	t.Run("pass valid namespace", func(t *testing.T) {
		assert.NoError(t, model.ValidateKeychainNamespace("@blue"))
		assert.NoError(t, model.ValidateKeychainNamespace("@@orange")) // Not prohibit double @
	})

	t.Run("fail invalid namespace", func(t *testing.T) {
		assert.ErrorIs(t, model.ValidateKeychainNamespace("blue"), model.ErrKeychainInvalidNamespace)
		assert.ErrorIs(t, model.ValidateKeychainNamespace("@"), model.ErrKeychainInvalidNamespace)
	})

	t.Run("bind prefix and namespace", func(t *testing.T) {
		assert.Equal(t, "magic.blue", model.KeychainNamespace("magic.", "@blue"))
	})
}
