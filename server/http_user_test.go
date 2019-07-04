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
	"github.com/mattermost/mattermost-plugin-jira/server/teststore"
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
				HasUpstream: true,
				UpstreamURL: teststore.UpstreamB_URL,
			},
		},
		"user B": {
			mattermostUserId:   teststore.UserB_MattermostId,
			expectedStatusCode: http.StatusOK,
			expectedResponse: jira.GetUserInfoResponse{
				UpstreamUserId: teststore.UserB_UpstreamId,
				HasUpstream:    true,
				UpstreamURL:    teststore.UpstreamB_URL,
				IsConnected:    true,
				Settings: upstream.UserSettings{
					Notifications: true,
				},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			api := &plugintest.API{}
			p := teststore.SetupTestPlugin(t, api, context.Config{
				EnableJiraUI: true,
			}, nil)
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
			expectedResponse: jira.GetSettingsInfoResponse{
				UIEnabled: true,
			},
		},
		"user A disabled": {
			enabled:            false,
			mattermostUserId:   teststore.UserA_MattermostId,
			expectedStatusCode: http.StatusOK,
			expectedResponse:   jira.GetSettingsInfoResponse{},
		},
	} {
		t.Run(name, func(t *testing.T) {
			api := &plugintest.API{}
			p := teststore.SetupTestPlugin(t, api, context.Config{
				EnableJiraUI: tc.enabled,
			}, nil)
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
