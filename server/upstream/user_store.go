// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package upstream

import (
	"fmt"
	"crypto/md5"
	
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-jira/server/store"
)

type UserStore interface {
	StoreUser(mattermostUserId string, user User) error
	DeleteUser(mattermostUserId, userKey string) error
	LoadUser(mattermostUserId string) (User, error)
	LoadMattermostUserId(userKey string) (string, error)
}

type LoadUserFunc func(data []byte) (User, error)

func (up upstream) StoreUser(mattermostUserId string, user User) error {
	mmkey := up.userkey(mattermostUserId)
	upkey := up.userkey(user.Key())
	err := store.StoreJSON(up.store, mmkey, user)
	if err != nil {
		return errors.WithMessagef(err, "failed to store upstream user for %q", mattermostUserId)
	}
	err = store.StoreJSON(up.store, upkey, mattermostUserId)
	if err != nil {
		return errors.WithMessagef(err, "failed to store mattermost Id for upstream user %q", user.DisplayName())
	}
	return nil
}

func (up upstream) LoadUser(mattermostUserId string) (User, error) {
	mmkey := up.userkey(mattermostUserId)
	data, err := up.store.Load(mmkey)
	if err != nil {
		return nil, errors.WithMessagef(err,
			"failed to load upstream user for: %q", mattermostUserId)
	}

	return up.loadUserFunc(data)
}

func (up upstream) LoadMattermostUserId(upstreamUserKey string) (string, error) {
	upkey := up.userkey(upstreamUserKey)
	mattermostUserId := ""
	err := store.LoadJSON(up.store, upkey, &mattermostUserId)
	if err != nil {
		return "", errors.WithMessagef(err,
			"failed to load Mattermost user ID for Jira user: %q", upstreamUserKey)
	}
	return mattermostUserId, nil
}

func (up upstream) DeleteUser(mattermostUserId, upstreamUserKey string) error {
	mmkey := up.userkey(mattermostUserId)
	upkey := up.userkey(upstreamUserKey)
	err := up.store.Delete(mmkey)
	if err != nil {
		return err
	}
	err = up.store.Delete(upkey)
	if err != nil {
		return err
	}
	return nil
}

func (up upstream) userkey(key string) string {
	if disablePrefixForUpstream {
		h := md5.New()
		fmt.Fprintf(h, "%s/%s", up.Config().Key, key)
		key = fmt.Sprintf("%x", h.Sum(nil))
	}
	return key
}