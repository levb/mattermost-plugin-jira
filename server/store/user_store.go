// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package store

import (
	"github.com/andygrunwald/go-jira"
	"github.com/pkg/errors"
)

type User struct {
	jira.User

	Oauth1AccessToken  string `json:",omitempty"`
	Oauth1AccessSecret string `json:",omitempty"`

	Settings *UserSettings
}

type UserSettings struct {
	Notifications bool `json:"notifications"`
}

type UserStore interface {
	Store(mattermostUserId string, user *User) error
	Delete(mattermostUserId string) error
	Load(mattermostUserId string) (*User, error)
	LoadMattermostUserId(upstreamUserKey string) (string, error)
}

type userStore struct {
	store Store
}

func NewUserStore(s Store) UserStore {
	return &userStore{s}
}

func (s userStore) Store(mattermostUserId string, user *User) error {
	err := StoreJSON(s.store, mattermostUserId, user)
	if err != nil {
		return err
	}
	err = StoreJSON(s.store, user.Name, mattermostUserId)
	if err != nil {
		return err
	}
	return nil
}

func (s userStore) Load(mattermostUserId string) (*User, error) {
	user := &User{}
	err := LoadJSON(s.store, mattermostUserId, &user)
	if err != nil {
		return user, errors.WithMessagef(err,
			"failed to load Jira user for user ID: %q", mattermostUserId)
	}
	return user, nil
}

func (s userStore) LoadMattermostUserId(upstreamUserKey string) (string, error) {
	mattermostUserId := ""
	err := LoadJSON(s.store, upstreamUserKey, &mattermostUserId)
	if err != nil {
		return "", errors.WithMessagef(err,
			"failed to load Mattermost user ID for Jira user: %q", upstreamUserKey)
	}
	if len(mattermostUserId) == 0 {
		return "", ErrNotFound
	}
	return mattermostUserId, nil
}

func (s userStore) Delete(mattermostUserId string) error {
	user, err := s.Load(mattermostUserId)
	if err != nil {
		return err
	}
	err = s.store.Delete(mattermostUserId)
	if err != nil {
		return err
	}
	err = s.store.Delete(user.Name)
	if err != nil {
		return err
	}
	return nil
}
