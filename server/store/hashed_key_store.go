// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package store

import (
	"crypto/md5"
	"fmt"
)

type hashedKeyStore struct {
	store  Store
	prefix string
}

var _ Store = (*hashedKeyStore)(nil)

func NewHashedKeyStore(s Store, prefix string) Store {
	return &hashedKeyStore{
		store:  s,
		prefix: prefix,
	}
}

func (s hashedKeyStore) Load(key string) ([]byte, error) {
	return s.store.Load(hashKey(s.prefix, key))
}

func (s hashedKeyStore) Store(key string, data []byte) error {
	return s.store.Store(hashKey(s.prefix, key), data)
}

func (s hashedKeyStore) Delete(key string) error {
	return s.store.Delete(hashKey(s.prefix, key))
}

func (s hashedKeyStore) Ensure(key string, value []byte) ([]byte, error) {
	return s.store.Ensure(hashKey(s.prefix, key), value)
}

func hashKey(prefix, hashableKey string) string {
	if hashableKey == "" {
		return prefix
	}

	h := md5.New()
	_, _ = h.Write([]byte(hashableKey))
	return fmt.Sprintf("%s%x", prefix, h.Sum(nil))
}
