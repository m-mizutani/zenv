package usecase

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/m-mizutani/goerr"
	"github.com/m-mizutani/zenv/pkg/domain/model"
	"github.com/m-mizutani/zenv/pkg/domain/types"
	"github.com/m-mizutani/zenv/pkg/utils"
)

func (x *Usecase) WriteSecret(input *model.WriteSecretInput) error {
	if err := input.Namespace.Validate(); err != nil {
		return err
	}
	if err := input.Key.Validate(); err != nil {
		return err
	}

	value := x.client.Prompt("Value")
	if value == "" {
		utils.Logger.Warn().Msg("No value provided, abort")
		return nil
	}

	envvar := &model.EnvVar{
		Key:    input.Key,
		Value:  types.EnvValue(value),
		Secret: true,
	}
	namespace := input.Namespace.ToNamespace(x.config.KeychainNamespacePrefix)
	if err := x.client.PutKeyChainValues([]*model.EnvVar{envvar}, namespace); err != nil {
		return goerr.Wrap(err).With("namespace", namespace).With("key", input.Key)
	}

	return nil
}

func (x *Usecase) GenerateSecret(input *model.GenerateSecretInput) error {
	if err := input.Namespace.Validate(); err != nil {
		return err
	}
	if err := input.Key.Validate(); err != nil {
		return err
	}
	if input.Length < 1 || 65335 < input.Length {
		return goerr.Wrap(types.ErrInvalidArgument, "variable length must be between 1 and 65335")
	}

	value, err := genRandomSecret(uint(input.Length))
	if err != nil {
		return err
	}

	envvar := &model.EnvVar{
		Key:    input.Key,
		Value:  types.EnvValue(value),
		Secret: true,
	}
	namespace := input.Namespace.ToNamespace(x.config.KeychainNamespacePrefix)
	if err := x.client.PutKeyChainValues([]*model.EnvVar{envvar}, namespace); err != nil {
		return goerr.Wrap(err).With("namespace", namespace).With("key", input.Key)
	}

	return nil
}

func genRandomSecret(n uint) (string, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return "", types.ErrGenerateRandom.Wrap(err)
	}
	return base64.URLEncoding.EncodeToString(b)[:n], nil
}

func (x *Usecase) ListNamespaces() error {
	namespaces, err := x.client.ListKeyChainNamespaces(x.config.KeychainNamespacePrefix)
	if err != nil {
		return err
	}

	for i := range namespaces {
		ns := namespaces[i].ToSuffix(x.config.KeychainNamespacePrefix)
		x.client.Stdout("%s\n", ns)
	}

	return nil
}

func (x *Usecase) DeleteSecret(input *model.DeleteSecretInput) error {
	ns := input.Namespace.ToNamespace(x.config.KeychainNamespacePrefix)
	if err := x.client.DeleteKeyChainValue(ns, input.Key); err != nil {
		return err
	}

	x.client.Stdout("%s %s is deleted\n", input.Key, input.Namespace)

	return nil
}

func (x *Usecase) ExportSecret(input *model.ExportSecretInput) error {
	prefix := x.config.KeychainNamespacePrefix

	var namespaces []types.Namespace
	if len(input.Namespaces) > 0 {
		for _, ns := range input.Namespaces {
			namespaces = append(namespaces, ns.ToNamespace(prefix))
		}
	} else {
		resp, err := x.client.ListKeyChainNamespaces(prefix)
		if err != nil {
			return err
		}
		namespaces = resp
	}

	var varSet []*model.NamespaceVars
	for _, ns := range namespaces {
		vars, err := x.client.GetKeyChainValues(ns)
		if err != nil {
			return err
		}

		varSet = append(varSet, &model.NamespaceVars{
			Namespace: ns.ToSuffix(prefix),
			Vars:      vars,
		})
	}

	raw, err := json.Marshal(varSet)
	if err != nil {
		return goerr.Wrap(err, "fail to marshal backup")
	}

	passphrase := types.Passphrase(x.client.Prompt("input passphrase"))
	key := sha256.Sum256([]byte(passphrase))
	c, err := aes.NewCipher(key[:])
	if err != nil {
		return goerr.Wrap(err, "failed to create cipher")
	}

	var w io.Writer
	var dstName string
	switch input.Output {
	case "-":
		w = os.Stdout
		dstName = "stdout"

	case "":
		fd, err := os.CreateTemp("./", "zenv-backup-*.json")
		if err != nil {
			return goerr.Wrap(err)
		}
		defer fd.Close() // #nosec, ignore even if Close() failed
		w = fd
		dstName = fd.Name()

	default:
		fd, err := os.Create(filepath.Clean(string(input.Output)))
		if err != nil {
			return goerr.Wrap(err)
		}
		defer fd.Close() // #nosec, ignore even if Close() failed
		w = fd
		dstName = string(input.Output)
	}

	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return goerr.Wrap(err, "failed to read random")
	}

	enc := cipher.NewCBCEncrypter(c, iv)

	paddingSize := (enc.BlockSize() - (len(raw) % enc.BlockSize())) % enc.BlockSize()
	plain := append(raw, bytes.Repeat([]byte{0}, paddingSize)...)
	encrypted := make([]byte, len(plain))
	enc.CryptBlocks(encrypted, plain)

	backup := &model.Backup{
		CreatedAt: time.Now(),
		Encrypted: encrypted,
		IV:        iv,
	}
	if err := json.NewEncoder(w).Encode(backup); err != nil {
		return goerr.Wrap(err, "failed to output backup stream")
	}

	x.client.Stdout("exported secrets to %s\n", dstName)

	return nil
}

func (x *Usecase) ImportSecret(input *model.ImportSecretInput) error {
	var r io.Reader
	var srcName string
	switch input.Input {
	case "-":
		r = os.Stdin
		srcName = "stdin"

	default:
		fd, err := os.Open(filepath.Clean(string(input.Input)))
		if err != nil {
			return goerr.Wrap(err)
		}
		defer fd.Close() // #nosec, ignore even if Close() failed
		r = fd
		srcName = string(input.Input)
	}

	var backup *model.Backup
	if err := json.NewDecoder(r).Decode(&backup); err != nil {
		return goerr.Wrap(err, "failed to unmarshal backup data")
	}

	passphrase := types.Passphrase(x.client.Prompt("input passphrase"))
	key := sha256.Sum256([]byte(passphrase))
	c, err := aes.NewCipher(key[:])
	if err != nil {
		return goerr.Wrap(err, "failed to create cipher")
	}

	decrypted := make([]byte, len(backup.Encrypted))
	dec := cipher.NewCBCDecrypter(c, backup.IV)
	dec.CryptBlocks(decrypted, backup.Encrypted)

	decrypted = bytes.TrimFunc(decrypted, func(r rune) bool {
		return r == 0
	})

	var vars []*model.NamespaceVars
	if err := json.Unmarshal(decrypted, &vars); err != nil {
		return goerr.Wrap(err, "fail to unmarshal decrypted data")
	}

	for _, v := range vars {
		ns := v.Namespace.ToNamespace(x.config.KeychainNamespacePrefix)
		if err := x.client.PutKeyChainValues(v.Vars, ns); err != nil {
			return err
		}
	}

	x.client.Stdout("imported secrets from %s\n", srcName)
	return nil
}
