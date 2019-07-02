// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package jira

import (
	"encoding/json"
	"net/http"

	"github.com/andygrunwald/go-jira"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-jira/server/action"
	"github.com/mattermost/mattermost-plugin-jira/server/kvstore"
	"github.com/mattermost/mattermost-plugin-jira/server/proxy"
	"github.com/mattermost/mattermost-plugin-jira/server/upstream"
)

// wrap jira.User into a different type name to avoid conflicts
type JiraUser jira.User

type User struct {
	upstream.BasicUser
	JiraUser
}

func (u User) UpstreamUserId() string {
	return u.JiraUser.AccountID
}

func UnmarshalUser(data []byte, defaultId string) (upstream.User, error) {
	u := User{}
	err := json.Unmarshal(data, &u)
	if err != nil {
		return nil, err
	}
	if u.BasicUser.MUserId == "" {
		u.BasicUser.MUserId = defaultId
	}
	if u.BasicUser.UUserId == "" {
		u.BasicUser.UUserId = u.JiraUser.AccountID
	}
	return &u, nil
}

// getUserConnectURL is a convenience function that checks that the user doesn't
// already exist before calling upstream's GetUserConnectURL.
func getUserConnectURL(pluginURL string, oneTimeStore kvstore.OneTimeStore,
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
	// Including the upstream BasicUser object here as an interface,
	// so it serializes itself inline with the other fields
	UpstreamUserId    string                `json:"upstream_user_id"`
	Settings          upstream.UserSettings `json:"settings"`
	IsConnected       bool                  `json:"is_connected"`
	UpstreamInstalled bool                  `json:"instance_installed"`
	UpstreamURL       string                `json:"jira_url,omitempty"`
}

func getUserInfo(upstore upstream.Store, mattermostUserId string) GetUserInfoResponse {
	resp := GetUserInfoResponse{}
	up, err := upstore.LoadCurrent()
	if err != nil {
		return resp
	}
	resp.UpstreamInstalled = true
	resp.UpstreamURL = up.Config().URL
	u, err := up.LoadUser(mattermostUserId)
	if err == nil {
		resp.IsConnected = true
		resp.UpstreamUserId = u.UpstreamUserId()
		resp.Settings = *u.Settings()
	}
	return resp
}

type GetSettingsInfoResponse struct {
	UIEnabled bool `json:"ui_enabled"`
}

func getSettingsInfo(enableJiraUI bool) GetSettingsInfoResponse {
	return GetSettingsInfoResponse{
		UIEnabled: enableJiraUI,
	}
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

func RequireClient(a action.Action) error {
	ac := a.Context()
	if ac.JiraClient != nil {
		return nil
	}
	err := action.Script{proxy.RequireUpstream, proxy.RequireUpstreamUser}.Run(a)
	if err != nil {
		return err
	}

	client, err := ac.Upstream.GetClient(ac.PluginId, ac.UpstreamUser)
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err)
	}
	ac.JiraClient = client
	a.Debugf("action: loaded upstream client for %q", ac.UpstreamUser.UpstreamUserId())
	return nil
}
