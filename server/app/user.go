// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package app

import (
	"net/http"

	"github.com/mattermost/mattermost-plugin-jira/server/instance"
	"github.com/mattermost/mattermost-plugin-jira/server/instance/loader"
	"github.com/mattermost/mattermost-plugin-jira/server/store"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

const (
	WS_EVENT_CONNECT    = "connect"
	WS_EVENT_DISCONNECT = "disconnect"
)

type GetUserInfoResponse struct {
	store.User
	IsConnected       bool   `json:"is_connected"`
	InstanceInstalled bool   `json:"instance_installed"`
	JIRAURL           string `json:"jira_url,omitempty"`
}

func GetUserConnectURL(
	userStore store.UserStore,
	oneTimeStore store.OneTimeStore,
	instance instance.Instance,
	pluginURL string,
	mattermostUserId string,
) (string, int, error) {

	// Users shouldn't be able to make multiple connections.
	user, err := userStore.Load(mattermostUserId)
	if err == nil && len(user.Key) != 0 {
		return "", http.StatusUnauthorized,
			errors.New("Already connected to a Jira account. Please use /jira disconnect to disconnect.")
	}

	redirectURL, err := instance.GetUserConnectURL(oneTimeStore, pluginURL, mattermostUserId)
	if err != nil {
		return "", http.StatusInternalServerError, err
	}

	return redirectURL, 0, nil
}

func GetUserInfo(
	instanceLoader loader.InstanceLoader,
	userStore store.UserStore,
	mattermostUserId string,
) GetUserInfoResponse {

	resp := GetUserInfoResponse{}
	instance, err := instanceLoader.Current()
	if err == nil {
		resp.InstanceInstalled = true
		resp.JIRAURL = instance.GetURL()
		user, err := userStore.Load(mattermostUserId)
		if err == nil {
			resp.User = *user
			resp.IsConnected = true
		}
	}
	return resp
}

func StoreUserNotify(api plugin.API, userStore store.UserStore, instance instance.Instance,
	mattermostUserId string, user *store.User) error {

	err := userStore.Store(mattermostUserId, user)
	if err != nil {
		return err
	}

	api.PublishWebSocketEvent(
		WS_EVENT_CONNECT,
		map[string]interface{}{
			"is_connected":  true,
			"jira_username": user.Name,
			"jira_url":      instance.GetURL(),
		},
		&model.WebsocketBroadcast{UserId: mattermostUserId},
	)

	return nil
}

func DeleteUserNotify(
	api plugin.API,
	userStore store.UserStore,
	mattermostUserId string,
) error {
	err := userStore.Delete(mattermostUserId)
	if err != nil {
		return err
	}

	api.PublishWebSocketEvent(
		WS_EVENT_DISCONNECT,
		map[string]interface{}{
			"is_connected": false,
		},
		&model.WebsocketBroadcast{UserId: mattermostUserId},
	)

	return nil
}

func StoreUserSettingsNotifications(userStore store.UserStore, mattermostUserId string,
	user *store.User, value bool) error {

	if user.Settings == nil {
		user.Settings = &store.UserSettings{}
	}
	user.Settings.Notifications = value
	err := userStore.Store(mattermostUserId, user)
	if err != nil {
		return errors.WithMessage(err, "Could not store new settings. Please contact your system administrator")
	}
	return nil
}
