// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package kvstore

import (
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin/plugintest"
	"github.com/mattermost/mattermost-server/plugin/plugintest/mock"
)

type MockedStore struct {
	KVStore
	plugintest.API
	Values map[string][]byte
}

func NewMockedStore() KVStore {
	s := MockedStore{
		Values: map[string][]byte{},
	}

	get := func(key string) []byte {
		return s.Values[key]
	}

	set := func(key string, value []byte) *model.AppError {
		s.Values[key] = value
		return nil
	}

	del := func(key string) *model.AppError {
		delete(s.Values, key)
		return nil
	}

	s.API.On("KVGet", mock.AnythingOfType("string")).Return(get, nil)
	s.API.On("KVSet", mock.AnythingOfType("string"), mock.AnythingOfType("[]uint8")).Return(set)
	s.API.On("KVDelete", mock.AnythingOfType("string")).Return(del)

	s.KVStore = NewPluginStore(&s.API)
	return s
}
