//go:build darwin
// +build darwin

package infra_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/m-mizutani/zenv/pkg/domain/model"
	"github.com/m-mizutani/zenv/pkg/domain/types"
	"github.com/m-mizutani/zenv/pkg/infra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeychainOSX(t *testing.T) {
	client := infra.New()
	ns := types.Namespace("zenv-test-" + uuid.NewString())
	v1 := &model.EnvVar{
		Key:    types.EnvKey("blue"),
		Value:  types.EnvValue("magic"),
		Secret: true,
	}
	v2 := &model.EnvVar{
		Key:    types.EnvKey("orange"),
		Value:  types.EnvValue("doll"),
		Secret: true,
	}
	t.Run("no item found before write", func(t *testing.T) {
		resp, err := client.GetKeyChainValues(ns)
		assert.ErrorIs(t, err, types.ErrKeychainNotFound)
		assert.Len(t, resp, 0)
	})

	t.Run("no item deleted before write", func(t *testing.T) {
		err := client.DeleteKeyChainValue(ns, v1.Key)
		assert.ErrorIs(t, err, types.ErrKeychainNotFound)
	})

	t.Run("write values", func(t *testing.T) {
		require.NoError(t, client.PutKeyChainValues([]*model.EnvVar{v1, v2}, ns))

		vars, err := client.GetKeyChainValues(ns)
		require.NoError(t, err)
		require.Len(t, vars, 2)
		assert.Contains(t, vars, v1)
		assert.Contains(t, vars, v2)
	})

	t.Run("delete values", func(t *testing.T) {
		require.NoError(t, client.DeleteKeyChainValue(ns, v1.Key))

		{
			vars, err := client.GetKeyChainValues(ns)
			require.NoError(t, err)
			require.Len(t, vars, 1)
			assert.NotContains(t, vars, v1)
			assert.Contains(t, vars, v2)
		}

		require.NoError(t, client.DeleteKeyChainValue(ns, v2.Key))
		{
			_, err := client.GetKeyChainValues(ns)
			assert.ErrorIs(t, err, types.ErrKeychainNotFound)
		}

		require.ErrorIs(t, client.DeleteKeyChainValue(ns, v1.Key), types.ErrKeychainNotFound)
	})
}
