// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package user

import (
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-jira/server/store"
)

type Store interface {
	Store(mattermostUserId string, user User) error
	Delete(mattermostUserId, userKey string) error
	Load(mattermostUserId string, userRef interface{}) error
	LoadMattermostUserId(userKey string) (string, error)
}

type userStore struct {
	store store.Store
}

func New(s store.Store) Store {
	return &userStore{s}
}

func (s userStore) Store(mattermostUserId string, user User) error {
	err := store.StoreJSON(s.store, mattermostUserId, user)
	if err != nil {
		return err
	}
	err = store.StoreJSON(s.store, user.Key(), mattermostUserId)
	if err != nil {
		return err
	}
	return nil
}

func (s userStore) Load(mattermostUserId string, userRef interface{}) error {
	err := store.LoadJSON(s.store, mattermostUserId, userRef)
	if err != nil {
		return errors.WithMessagef(err,
			"failed to load Jira user for user ID: %q", mattermostUserId)
	}
	return nil
}

func (s userStore) LoadMattermostUserId(upstreamUserKey string) (string, error) {
	mattermostUserId := ""
	err := store.LoadJSON(s.store, upstreamUserKey, &mattermostUserId)
	if err != nil {
		return "", errors.WithMessagef(err,
			"failed to load Mattermost user ID for Jira user: %q", upstreamUserKey)
	}
	if len(mattermostUserId) == 0 {
		return "", store.ErrNotFound
	}
	return mattermostUserId, nil
}

func (s userStore) Delete(mattermostUserId, userKey string) error {
	err := s.store.Delete(mattermostUserId)
	if err != nil {
		return err
	}
	err = s.store.Delete(userKey)
	if err != nil {
		return err
	}
	return nil
}
