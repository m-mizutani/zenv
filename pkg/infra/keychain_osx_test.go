//go:build darwin
// +build darwin

package infra_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/zenv/pkg/domain/model"
	"github.com/m-mizutani/zenv/pkg/domain/types"
	"github.com/m-mizutani/zenv/pkg/infra"
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
		gt.R1(client.GetKeyChainValues(ns)).Error(t).Is(types.ErrKeychainNotFound)
	})

	t.Run("no item deleted before write", func(t *testing.T) {
		gt.Error(t, client.DeleteKeyChainValue(ns, v1.Key)).Is(types.ErrKeychainNotFound)
	})

	t.Run("write values", func(t *testing.T) {
		gt.NoError(t, client.PutKeyChainValues([]*model.EnvVar{v1, v2}, ns))
		vars := gt.R1(client.GetKeyChainValues(ns)).NoError(t)
		gt.Array(t, vars).
			Length(2).
			Have(v1).
			Have(v2)
	})

	t.Run("delete values", func(t *testing.T) {
		gt.NoError(t, client.DeleteKeyChainValue(ns, v1.Key))

		{
			vars := gt.R1(client.GetKeyChainValues(ns)).NoError(t)
			gt.Array(t, vars).
				Length(1).
				NotHave(v1).
				Have(v2)
		}

		gt.NoError(t, client.DeleteKeyChainValue(ns, v2.Key))
		gt.R1(client.GetKeyChainValues(ns)).
			Error(t).Is(types.ErrKeychainNotFound)
		gt.Error(t, client.DeleteKeyChainValue(ns, v1.Key)).
			Is(types.ErrKeychainNotFound)
	})
}
