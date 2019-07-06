package proxy

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-jira/server/kvstore"
)

func EnsureRSAPrivateKey(kv kvstore.KVStore) (*rsa.PrivateKey, error) {
	// Ensure we generate the secrets on first start
	rsaPrivateKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to generate private key")
	}
	rsaPrivateKeyBytes, err := json.Marshal(rsaPrivateKey)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to marshal private key")
	}
	newRSAPrivateKeyBytes, err := kvstore.Ensure(kv, kvstore.KeyRSAPrivateKey, rsaPrivateKeyBytes)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to generate private key")
	}
	newRSAPrivateKey := &rsa.PrivateKey{}
	err = json.Unmarshal(newRSAPrivateKeyBytes, newRSAPrivateKey)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to unmarshal private key")
	}
	return rsaPrivateKey, nil
}

func EnsureAuthTokenSecret(kv kvstore.KVStore) ([]byte, error) {
	// Ensure we generate the secrets on first start
	secret := make([]byte, 32)
	_, err := rand.Reader.Read(secret)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to generate token secret")
	}
	return kvstore.Ensure(kv, kvstore.KeyTokenSecret, secret)
}
