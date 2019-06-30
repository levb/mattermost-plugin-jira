// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package lib

import (
	"net/http"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-jira/server/kvstore"
	"github.com/mattermost/mattermost-plugin-jira/server/upstream"
)

// GetUserConnectURL is a convenience function that checks that the user doesn't
// already exist before calling upstream's GetUserConnectURL.
func GetUserConnectURL(pluginURL string, oneTimeStore kvstore.OneTimeStore,
	up upstream.Upstream, mattermostUserId string) (string, int, error) {
	// Users shouldn't be able to make multiple connections.
	_, err := up.LoadUser(mattermostUserId)
	switch err {
	case nil:
		return "", http.StatusUnauthorized,
			errors.New("Already connected to a Jira account. Please use /jira disconnect to disconnect.")

	case kvstore.ErrNotFound:

	default:
		return "", http.StatusInternalServerError, err
	}

	redirectURL, err := up.GetUserConnectURL(oneTimeStore, pluginURL, mattermostUserId)
	if err != nil {
		return "", http.StatusInternalServerError, err
	}

	return redirectURL, 0, nil
}

type GetUserInfoResponse struct {
	// Including the upstream User object here as an interface,
	// so it serializes itself inline with the other fields
	upstream.User

	IsConnected       bool   `json:"is_connected"`
	UpstreamInstalled bool   `json:"instance_installed"`
	UpstreamURL       string `json:"jira_url,omitempty"`
}

func GetUserInfo(upstore upstream.Store, mattermostUserId string) GetUserInfoResponse {
	resp := GetUserInfoResponse{}
	up, err := upstore.LoadCurrent()
	if err != nil {
		return resp
	}
	resp.UpstreamInstalled = true
	resp.UpstreamURL = up.Config().URL
	resp.User, err = up.LoadUser(mattermostUserId)
	if err == nil {
		resp.IsConnected = true
	}

	return resp
}

func StoreUserSettingsNotifications(up upstream.Upstream, u upstream.User, value bool) error {
	settings := u.Settings()
	settings.Notifications = value
	err := up.StoreUser(u)
	if err != nil {
		return errors.WithMessage(err, "Could not store new settings. Please contact your system administrator")
	}
	return nil
}
