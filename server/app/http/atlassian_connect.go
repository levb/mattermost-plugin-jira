// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package http

import (
	"github.com/mattermost/mattermost-plugin-jira/server/action"
	"github.com/mattermost/mattermost-plugin-jira/server/app"
	"github.com/mattermost/mattermost-plugin-jira/server/instance/jira_cloud"
)

func getACJSON(a action.Action) error {
	err := action.Script{
		app.RequireHTTPGet,
	}.Run(a)
	if err != nil {
		return err
	}
	ac := a.Context()

	return action.HTTPRespondTemplate(a,
		"application/json",
		map[string]string{
			"BaseURL":                      ac.PluginURL,
			"RouteACJSON":                  routeACJSON,
			"RouteACInstalled":             routeACInstalled,
			"RouteACUninstalled":           routeACUninstalled,
			"RouteACUserRedirectWithToken": routeACUserRedirectWithToken,
			"UserRedirectPageKey":          jira_cloud.UserLandingPageKey,
			"ExternalURL":                  ac.MattermostSiteURL,
			"PluginKey":                    ac.PluginKey,
		})
}

func processACInstalled(a action.Action) error {
	err := action.Script{
		app.RequireHTTPPost,
	}.Run(a)
	if err != nil {
		return err
	}
	ac := a.Context()
	request, err := action.HTTPRequest(a)
	if err != nil {
		return err
	}

	status, err := app.ProcessACInstalled(ac.API,
		ac.InstanceStore, ac.CurrentInstanceStore, ac.AuthTokenSecret, request.Body)
	if err != nil {
		return a.RespondError(status, err,
			"failed to process atlassian-connect installed event")
	}

	return a.RespondJSON([]string{"OK"})
}
