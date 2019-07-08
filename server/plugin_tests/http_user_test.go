// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package plugin_tests

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-plugin-jira/server/jira"
	"github.com/mattermost/mattermost-plugin-jira/server/plugin"
	"github.com/mattermost/mattermost-plugin-jira/server/upstream"
	mmplugin "github.com/mattermost/mattermost-server/plugin"
	"github.com/mattermost/mattermost-server/plugin/plugintest"
)

func TestGetUserInfo(t *testing.T) {

	for name, tc := range map[string]struct {
		rawStore           bool
		mattermostUserId   string
		expectedStatusCode int
		expectedResponse   jira.GetUserInfoResponse
	}{
		"not found": {
			rawStore:           true,
			mattermostUserId:   KeyDoesNotExist,
			expectedStatusCode: http.StatusOK,
			expectedResponse:   jira.GetUserInfoResponse{},
		},
		"no user": {
			mattermostUserId:   "",
			expectedStatusCode: http.StatusUnauthorized,
			expectedResponse:   jira.GetUserInfoResponse{},
		},
		"user A": {
			mattermostUserId:   UserA_MattermostId,
			expectedStatusCode: http.StatusOK,
			expectedResponse: jira.GetUserInfoResponse{
				HasUpstream: true,
				UpstreamURL: UpstreamB_URL,
			},
		},
		"user B": {
			mattermostUserId:   UserB_MattermostId,
			expectedStatusCode: http.StatusOK,
			expectedResponse: jira.GetUserInfoResponse{
				UpstreamUserId: UserB_UpstreamId,
				HasUpstream:    true,
				UpstreamURL:    UpstreamB_URL,
				IsConnected:    true,
				Settings: upstream.UserSettings{
					Notifications: true,
				},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			api := &plugintest.API{}
			p := SetupTestPlugin(t, api, plugin.Config{
				MainConfig: plugin.MainConfig{
					EnableJiraUI: false,
				},
			}, nil)

			if !tc.rawStore {
				Store2Upstreams2Users(t, p.Proxy)
			}
			w := httptest.NewRecorder()
			request := httptest.NewRequest("GET", "/api/v2/userinfo", nil)
			request.Header.Set("Mattermost-User-Id", tc.mattermostUserId)
			p.ServeHTTP(&mmplugin.Context{}, w, request)
			require.Equal(t, tc.expectedStatusCode, w.Result().StatusCode)
			if w.Result().StatusCode != http.StatusOK {
				return
			}
			resp := jira.GetUserInfoResponse{}
			data, err := ioutil.ReadAll(w.Result().Body)
			require.True(t, len(data) > 0)
			require.Nil(t, err)
			err = json.Unmarshal(data, &resp)
			require.Nil(t, err)
			require.Equal(t, tc.expectedResponse, resp)
		})
	}
}
func TestGetSettingsInfo(t *testing.T) {

	for name, tc := range map[string]struct {
		enabled            bool
		mattermostUserId   string
		expectedStatusCode int
		expectedResponse   jira.GetSettingsInfoResponse
	}{
		"no user": {
			enabled:            true,
			mattermostUserId:   "",
			expectedStatusCode: http.StatusUnauthorized,
			expectedResponse:   jira.GetSettingsInfoResponse{},
		},
		"user A": {
			enabled:            true,
			mattermostUserId:   UserA_MattermostId,
			expectedStatusCode: http.StatusOK,
			expectedResponse: jira.GetSettingsInfoResponse{
				UIEnabled: true,
			},
		},
		"user A disabled": {
			enabled:            false,
			mattermostUserId:   UserA_MattermostId,
			expectedStatusCode: http.StatusOK,
			expectedResponse:   jira.GetSettingsInfoResponse{},
		},
	} {
		t.Run(name, func(t *testing.T) {
			api := &plugintest.API{}
			p := SetupTestPlugin(t, api, plugin.Config{
				MainConfig: plugin.MainConfig{
					EnableJiraUI: tc.enabled,
				},
			}, nil)
			Store2Upstreams2Users(t, p.Proxy)

			w := httptest.NewRecorder()
			request := httptest.NewRequest("GET", "/api/v2/settingsinfo", nil)
			request.Header.Set("Mattermost-User-Id", tc.mattermostUserId)
			p.ServeHTTP(&mmplugin.Context{}, w, request)
			require.Equal(t, tc.expectedStatusCode, w.Result().StatusCode)
			if w.Result().StatusCode != http.StatusOK {
				return
			}
			resp := jira.GetSettingsInfoResponse{}
			data, err := ioutil.ReadAll(w.Result().Body)
			require.True(t, len(data) > 0)
			require.Nil(t, err)
			err = json.Unmarshal(data, &resp)
			require.Nil(t, err)
			require.Equal(t, tc.expectedResponse, resp)
		})
	}
}
