// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package store

import (
	"github.com/pkg/errors"
)

type EnsuredStore interface {
	Ensure(key string, valuef func() ([]byte, error)) ([]byte, error)
}

type ensuredStore struct {
	Store
}

func NewEnsuredStore(s Store) EnsuredStore {
	return &ensuredStore{s}
}

func (s ensuredStore) Ensure(key string, valuef func() ([]byte, error)) (secret []byte, returnErr error) {
	defer func() {
		if returnErr == nil {
			return
		}
		returnErr = errors.WithMessage(returnErr, "failed to ensure auth token secret")
	}()

	// nil, nil == NOT_FOUND, if we don't already have a key, try to generate one.
	secret, err := s.Store.Load(key)
	if err != nil {
		return nil, err
	}

	if len(secret) == 0 && valuef != nil {
		var newSecret []byte
		newSecret, err = valuef()
		if err != nil {
			return nil, err
		}
		err = s.Store.Store(key, newSecret)
		if err != nil {
			return nil, err
		}
		secret = newSecret
	}

	// If we weren't able to save a new key above, another server must have beat us to it. Get the
	// key from the database, and if that fails, error out.
	if secret == nil {
		secret, err = s.Load(key)
		if err != nil {
			return nil, err
		}
	}

	return secret, nil
}
