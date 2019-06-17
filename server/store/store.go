// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package store

import (
	"encoding/json"

	mmplugin "github.com/mattermost/mattermost-server/plugin"

	"github.com/pkg/errors"
)

type Store interface {
	Load(key string) ([]byte, error)
	Store(key string, data []byte) error
	Delete(key string) error
}

type pluginStore struct {
	api mmplugin.API
}

var _ Store = (*pluginStore)(nil)

var ErrNotFound = errors.New("not found")

func NewPluginStore(api mmplugin.API) Store {
	return &pluginStore{
		api: api,
	}
}

func (s pluginStore) Load(key string) ([]byte, error) {
	data, appErr := s.api.KVGet(key)
	if appErr != nil {
		return nil, errors.WithMessage(appErr, "failed plugin KVGet")
	}
	if data == nil {
		return nil, ErrNotFound
	}
	return data, nil
}

func (s pluginStore) Store(key string, data []byte) error {
	appErr := s.api.KVSet(key, data)
	if appErr != nil {
		return errors.WithMessagef(appErr, "failed plugin KVSet %q", key)
	}
	return nil
}

func (s pluginStore) Delete(key string) error {
	appErr := s.api.KVDelete(key)
	if appErr != nil {
		return errors.WithMessagef(appErr, "failed plugin KVdelete %q", key)
	}
	return nil
}

func LoadJSON(s Store, key string, v interface{}) (returnErr error) {
	data, err := s.Load(key)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

func StoreJSON(s Store, key string, v interface{}) (returnErr error) {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return s.Store(key, data)
}
