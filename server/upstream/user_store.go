// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package upstream

import (
	"crypto/md5"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-jira/server/kvstore"
	"github.com/mattermost/mattermost-server/model"
)

const (
	WebsocketEventConnect    = "connect"
	WebsocketEventDisconnect = "disconnect"
)

type UserStore interface {
	StoreUser(User) error
	DeleteUser(User) error
	StoreUserNotify(User) error
	DeleteUserNotify(User) error
	LoadUser(mattermostUserId string) (User, error)
	LoadMattermostUserId(upstreamUserId string) (string, error)
}

func (up Basic) StoreUser(u User) error {
	mmkey := up.userkey(u.MattermostUserId())
	upkey := up.userkey(u.UpstreamUserId())
	kv := up.Context().KVStore

	err := kvstore.StoreJSON(kv, mmkey, u)
	if err != nil {
		return err
	}
	err = kvstore.StoreJSON(kv, upkey, u.MattermostUserId())
	if err != nil {
		return err
	}
	return nil
}

func (up Basic) LoadUserRaw(mattermostUserId string) ([]byte, error) {
	mmkey := up.userkey(mattermostUserId)
	kv := up.Context().KVStore
	return kv.Load(mmkey)
}

func (up Basic) LoadUser(mattermostUserId string) (User, error) {
	data, err := up.LoadUserRaw(mattermostUserId)
	if err != nil {
		return nil, err
	}

	basic := BasicUser{}
	err = json.Unmarshal(data, &basic)
	if err != nil {
		return nil, err
	}
	if basic.MUserId != mattermostUserId {
		return nil, errors.Errorf(
			"stored user id %q did not match the current user id: %q", basic.MUserId, mattermostUserId)
	}

	return &basic, nil
}

func (up Basic) LoadMattermostUserId(upstreamUserId string) (string, error) {
	upkey := up.userkey(upstreamUserId)
	mattermostUserId := ""
	kv := up.Context().KVStore

	err := kvstore.LoadJSON(kv, upkey, &mattermostUserId)
	if err != nil {
		return "", err
	}
	return mattermostUserId, nil
}

func (up Basic) DeleteUser(u User) error {
	mmkey := up.userkey(u.MattermostUserId())
	upkey := up.userkey(u.UpstreamUserId())
	kv := up.Context().KVStore
	err := kv.Delete(mmkey)
	if err != nil {
		return err
	}
	err = kv.Delete(upkey)
	if err != nil {
		return err
	}
	return nil
}

func (up Basic) userkey(key string) string {
	h := md5.New()
	fmt.Fprintf(h, "%s/%s", up.UpstreamKey, key)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func (up Basic) StoreUserNotify(u User) error {
	err := up.StoreUser(u)
	if err != nil {
		return err
	}

	up.Context().API.PublishWebSocketEvent(
		WebsocketEventConnect,
		map[string]interface{}{
			"is_connected": true,
		},
		&model.WebsocketBroadcast{UserId: u.MattermostUserId()},
	)

	return nil
}

func (up Basic) DeleteUserNotify(u User) error {
	err := up.DeleteUser(u)
	if err != nil {
		return err
	}

	up.Context().API.PublishWebSocketEvent(
		WebsocketEventDisconnect,
		map[string]interface{}{
			"is_connected": false,
		},
		&model.WebsocketBroadcast{UserId: u.MattermostUserId()},
	)

	return nil
}
