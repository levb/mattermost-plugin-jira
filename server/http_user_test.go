// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-plugin-jira/server/context"
	"github.com/mattermost/mattermost-plugin-jira/server/jira"
	"github.com/mattermost/mattermost-plugin-jira/server/kvstore"
	"github.com/mattermost/mattermost-plugin-jira/server/plugin"
	"github.com/mattermost/mattermost-plugin-jira/server/teststore"
	"github.com/mattermost/mattermost-plugin-jira/server/upstream"
	"github.com/mattermost/mattermost-server/model"
	mmplugin "github.com/mattermost/mattermost-server/plugin"
	"github.com/mattermost/mattermost-server/plugin/plugintest"
	"github.com/mattermost/mattermost-server/plugin/plugintest/mock"
)

func setupTestPlugin(t *testing.T, conf context.Config) *plugin.Plugin {
	kv := kvstore.NewMockedStore()
	api := &plugintest.API{}
	p := &plugin.Plugin{}
	p.SetAPI(api)

	api.On("GetUserByUsername", mock.AnythingOfTypeArgument("string")).Return(&model.User{}, nil)
	api.On("LogDebug",
		mock.AnythingOfTypeArgument("string")).Return(nil)
	api.On("LogInfo",
		mock.AnythingOfTypeArgument("string")).Return(nil)

	f, err := plugin.MakeContext(p.API, kv, teststore.Unmarshallers, "pluginID", "version-string", "..")
	require.Nil(t, err)
	p.UpdateContext(f)
	p.UpdateContext(func(c *context.Context) {
		plugin.RefreshContext(p.API, c, context.Config{}, conf, "site.url", "")
	})

	return p
}

func TestGetUserInfo(t *testing.T) {

	for name, tc := range map[string]struct {
		rawStore           bool
		mattermostUserId   string
		expectedStatusCode int
		expectedResponse   jira.GetUserInfoResponse
	}{
		"not found": {
			rawStore:           true,
			mattermostUserId:   teststore.KeyDoesNotExist,
			expectedStatusCode: http.StatusOK,
			expectedResponse:   jira.GetUserInfoResponse{},
		},
		"no user": {
			mattermostUserId:   "",
			expectedStatusCode: http.StatusUnauthorized,
			expectedResponse:   jira.GetUserInfoResponse{},
		},
		"user A": {
			mattermostUserId:   teststore.UserA_MattermostId,
			expectedStatusCode: http.StatusOK,
			expectedResponse: jira.GetUserInfoResponse{
				UpstreamInstalled: true,
				UpstreamURL:       teststore.UpstreamB_URL,
			},
		},
		"user B": {
			mattermostUserId:   teststore.UserB_MattermostId,
			expectedStatusCode: http.StatusOK,
			expectedResponse: jira.GetUserInfoResponse{
				UpstreamUserId:    teststore.UserB_UpstreamId,
				UpstreamInstalled: true,
				UpstreamURL:       teststore.UpstreamB_URL,
				IsConnected:       true,
				Settings: upstream.UserSettings{
					Notifications: true,
				},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			p := setupTestPlugin(t, context.Config{
				EnableJiraUI: true,
			})
			if !tc.rawStore {
				teststore.UpstreamStore_2Upstreams2Users(t, p.GetContext().UpstreamStore)
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
			mattermostUserId:   teststore.UserA_MattermostId,
			expectedStatusCode: http.StatusOK,
			expectedResponse:   jira.GetSettingsInfoResponse{true},
		},
		"user A disabled": {
			enabled:            false,
			mattermostUserId:   teststore.UserA_MattermostId,
			expectedStatusCode: http.StatusOK,
			expectedResponse:   jira.GetSettingsInfoResponse{},
		},
	} {
		t.Run(name, func(t *testing.T) {
			p := setupTestPlugin(t, context.Config{
				EnableJiraUI: tc.enabled,
			})
			teststore.UpstreamStore_2Upstreams2Users(t, p.GetContext().UpstreamStore)

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
