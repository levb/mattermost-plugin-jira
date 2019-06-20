// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package store

import (
	"github.com/pkg/errors"

	mmplugin "github.com/mattermost/mattermost-server/plugin"
)

const (
	prefixOneTimeSecret = "ots_" // + unique key that will be deleted after the first verification
)

type OAuth1aTemporaryCredentials struct {
	Token  string
	Secret string
}

type OneTimeStore interface {
	Store
	StoreOauth1aTemporaryCredentials(mmUserId string, credentials *OAuth1aTemporaryCredentials) error
	LoadOauth1aTemporaryCredentials(mmUserId string) (*OAuth1aTemporaryCredentials, error)
}

type pluginOTSStore struct {
	pluginStore
	TTLSeconds int64
}

func NewPluginOneTimeStore(api mmplugin.API, ttlSeconds int64) OneTimeStore {
	return &pluginOTSStore{
		pluginStore: pluginStore{api},
		TTLSeconds:  ttlSeconds,
	}
}

func (s pluginOTSStore) Load(key string) (data []byte, returnErr error) {
	data, err := s.pluginStore.Load(key)

	_ = s.Delete(key)

	return data, err
}

func (s pluginOTSStore) Store(key string, data []byte) error {
	appErr := s.api.KVSetWithExpiry(key, data, s.TTLSeconds)
	if appErr != nil {
		return errors.WithMessage(appErr, "failed to store")
	}
	return nil
}

func (s pluginOTSStore) StoreOauth1aTemporaryCredentials(mmUserId string, credentials *OAuth1aTemporaryCredentials) error {
	return StoreJSON(s, hashKey(prefixOneTimeSecret, mmUserId), &credentials)
}

func (s pluginOTSStore) LoadOauth1aTemporaryCredentials(mmUserId string) (*OAuth1aTemporaryCredentials, error) {
	var credentials OAuth1aTemporaryCredentials
	err := LoadJSON(s, hashKey(prefixOneTimeSecret, mmUserId), &credentials)
	if err != nil {
		return nil, err
	}
	return &credentials, nil
}
