// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package upstream

import (
	"crypto/md5"
	"fmt"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-jira/server/kvstore"
)

func (up BasicUpstream) StoreUser(u User) error {
	mmkey := up.userkey(u.MattermostId())
	upkey := up.userkey(u.UpstreamId())

	err := kvstore.StoreJSON(up.kv, mmkey, u)
	if err != nil {
		return err
	}
	err = kvstore.StoreJSON(up.kv, upkey, u.MattermostId())
	if err != nil {
		return err
	}
	return nil
}

func (up BasicUpstream) LoadUser(mattermostUserId string) (User, error) {
	mmkey := up.userkey(mattermostUserId)

	fmt.Printf("<><> LoadUser 1 %+v\n", up)
	fmt.Printf("<><> LoadUser 2 %+v\n", up.kv)
	data, err := up.kv.Load(mmkey)
	if err != nil {
		return nil, err
	}

	u, err := up.unmarshaller.UnmarshalUser(data, mattermostUserId)
	if err != nil {
		return nil, err
	}
	if u.MattermostId() != mattermostUserId {
		return nil, errors.Errorf(
			"stored user id %q did not match the current user id: %q", u.MattermostId(), mattermostUserId)
	}

	return u, nil
}

func (up BasicUpstream) LoadMattermostUserId(upstreamUserId string) (string, error) {
	upkey := up.userkey(upstreamUserId)
	mattermostUserId := ""
	err := kvstore.LoadJSON(up.kv, upkey, &mattermostUserId)
	if err != nil {
		return "", err
	}
	return mattermostUserId, nil
}

func (up BasicUpstream) DeleteUser(u User) error {
	mmkey := up.userkey(u.MattermostId())
	upkey := up.userkey(u.UpstreamId())
	err := up.kv.Delete(mmkey)
	if err != nil {
		return err
	}
	err = up.kv.Delete(upkey)
	if err != nil {
		return err
	}
	return nil
}

func (up BasicUpstream) userkey(key string) string {
	h := md5.New()
	fmt.Fprintf(h, "%s/%s", up.Config().Key, key)
	return fmt.Sprintf("%x", h.Sum(nil))
}
