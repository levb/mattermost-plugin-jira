// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package jira

import (
	"net/http"

	"github.com/andygrunwald/go-jira"
	"github.com/mattermost/mattermost-plugin-jira/server/upstream"
	"github.com/mattermost/mattermost-plugin-jira/server/store"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

const (
	WebsocketEventConnect    = "connect"
	WebsocketEventDisconnect = "disconnect"
)

type User struct {
	jira.User
	upstream.UserSettings `json:"settings"`
	MattermostUserId string
}

type GetUserInfoResponse struct {
	User
	IsConnected       bool   `json:"is_connected"`
	UpstreamInstalled bool   `json:"instance_installed"`
	JIRAURL           string `json:"jira_url,omitempty"`
}

func (user *User) Settings() *upstream.UserSettings {
	return &user.UserSettings
}

func GetUserConnectURL(
	userStore upstream.UserStore,
	oneTimeStore store.OneTimeStore,
	up upstream.Upstream,
	pluginURL string,
	mattermostUserId string,
) (string, int, error) {
	juser := User{}

	// Users shouldn't be able to make multiple connections.
	err := userStore.Load(mattermostUserId, &juser)
	if err == nil {
		return "", http.StatusUnauthorized,
			errors.New("Already connected to a Jira account. Please use /jira disconnect to disconnect.")
	}

	redirectURL, err := upstream.GetUserConnectURL(oneTimeStore, pluginURL, mattermostUserId)
	if err != nil {
		return "", http.StatusInternalServerError, err
	}

	return redirectURL, 0, nil
}

func GetUserInfo(
	upstreamLoader loader.UpstreamLoader,
	userStore user.UserStore,
	mattermostUserId string,
) GetUserInfoResponse {

	resp := GetUserInfoResponse{}
	up, err := upstreamLoader.Current()
	if err == nil {
		resp.UpstreamInstalled = true
		resp.JIRAURL = upstream.GetURL()
		err := userStore.Load(mattermostUserId, &resp.User)
		if err == nil {
			resp.IsConnected = true
		}
	}
	return resp
}

func StoreUserNotify(api plugin.API, userStore user.UserStore, mattermostUserId string,
	user user.User) error {

	err := userStore.Store(mattermostUserId, user)
	if err != nil {
		return err
	}

	api.PublishWebSocketEvent(
		WebsocketEventConnect,
		map[string]interface{}{
			"is_connected": true,
		},
		&model.WebsocketBroadcast{UserId: mattermostUserId},
	)

	return nil
}

func DeleteUserNotify(
	api plugin.API,
	userStore user.UserStore,
	mattermostUserId, userKey string,
) error {
	err := userStore.Delete(mattermostUserId, userKey)
	if err != nil {
		return err
	}

	api.PublishWebSocketEvent(
		WebsocketEventDisconnect,
		map[string]interface{}{
			"is_connected": false,
		},
		&model.WebsocketBroadcast{UserId: mattermostUserId},
	)

	return nil
}

func StoreUserSettingsNotifications(userStore user.UserStore, mattermostUserId string,
	user user.User, value bool) error {

	settings := user.Settings()
	settings.Notifications = value
	err := userStore.Store(mattermostUserId, user)
	if err != nil {
		return errors.WithMessage(err, "Could not store new settings. Please contact your system administrator")
	}
	return nil
}