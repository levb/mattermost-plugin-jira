// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package upstream

import (
	"crypto/md5"
	"fmt"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-jira/server/store"
)

func (up upstream) StoreUser(u User) error {
	mmkey := up.userkey(u.MattermostId())
	upkey := up.userkey(u.UpstreamId())

	err := store.StoreJSON(up.store, mmkey, u)
	if err != nil {
		return errors.WithMessagef(err, "failed to store upstream user for %q", u.MattermostId())
	}
	err = store.StoreJSON(up.store, upkey, u.MattermostId())
	if err != nil {
		return errors.WithMessagef(err, "failed to store mattermost Id for upstream user %q", u.UpstreamDisplayName())
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

	u, err := up.unmarshaller.UnmarshalUser(data)
	if err != nil {
		return nil, errors.WithMessagef(err,
			"failed to unmarshal user for: %q", mattermostUserId)
	}
	if u.MattermostId() == "" && u.MattermostId() != mattermostUserId {
		return nil, errors.Errorf("stored user id mismatch: %q", mattermostUserId)
	}

	return u, nil
}

func (up upstream) LoadMattermostUserId(upstreamUserId string) (string, error) {
	upkey := up.userkey(upstreamUserId)
	mattermostUserId := ""
	err := store.LoadJSON(up.store, upkey, &mattermostUserId)
	if err != nil {
		return "", errors.WithMessagef(err,
			"failed to load Mattermost user ID for upstream user: %q", upstreamUserId)
	}
	return mattermostUserId, nil
}

func (up upstream) DeleteUser(u User) error {
	mmkey := up.userkey(u.MattermostId())
	upkey := up.userkey(u.UpstreamId())
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
