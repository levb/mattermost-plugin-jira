// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package kvstore

import (
	"github.com/mattermost/mattermost-server/plugin"

	"github.com/pkg/errors"
)

type pluginStore struct {
	api plugin.API
}

var _ KVStore = (*pluginStore)(nil)

func NewPluginStore(api plugin.API) KVStore {
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

