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
	mmkey := up.userkey(u.MattermostUserId())
	upkey := up.userkey(u.UpstreamUserId())

	err := kvstore.StoreJSON(up.kv, mmkey, u)
	if err != nil {
		return err
	}
	err = kvstore.StoreJSON(up.kv, upkey, u.MattermostUserId())
	if err != nil {
		return err
	}
	return nil
}

func (up BasicUpstream) LoadUser(mattermostUserId string) (User, error) {
	mmkey := up.userkey(mattermostUserId)
	data, err := up.kv.Load(mmkey)
	if err != nil {
		return nil, err
	}

	u, err := up.unmarshaller.UnmarshalUser(data, mattermostUserId)
	if err != nil {
		return nil, err
	}
	fmt.Printf("<><> LoadUser: %v %+v", err, u)
	if u.MattermostUserId() != mattermostUserId {
		return nil, errors.Errorf(
			"stored user id %q did not match the current user id: %q", u.MattermostUserId(), mattermostUserId)
	}

	return u, nil
}

func (up BasicUpstream) LoadMattermostUserId(upstreamUserId string) (string, error) {
	upkey := up.userkey(upstreamUserId)
	mattermostUserId := ""
	err := kvstore.LoadJSON(up.kv, upkey, &mattermostUserId)
	fmt.Printf("<><> LoadMattermostUserId: %v %q", err, mattermostUserId)
	if err != nil {
		return "", err
	}
	return mattermostUserId, nil
}

func (up BasicUpstream) DeleteUser(u User) error {
	mmkey := up.userkey(u.MattermostUserId())
	upkey := up.userkey(u.UpstreamUserId())
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
