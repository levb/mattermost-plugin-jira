// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package jira_server

import (
	"net/http"
	"path"

	"github.com/mattermost/mattermost-plugin-jira/server/action"
	"github.com/mattermost/mattermost-plugin-jira/server/action/http_action"
	"github.com/mattermost/mattermost-plugin-jira/server/jira"
	"github.com/mattermost/mattermost-plugin-jira/server/proxy"
)

const (
	routeOAuth1Complete  = "/oauth1/complete.html"
	routeOAuth1PublicKey = "/oauth1/public_key.html"
)

var HTTPRoutes = map[string]*action.Route{
	routeOAuth1Complete: action.NewRoute(
		proxy.RequireHTTPMethod(http.MethodGet),
		proxy.RequireMattermostUserId,
		proxy.RequireMattermostUser,
		proxy.RequireUpstream,
		RequireUpstream,
		httpCompleteOAuth1),
}

func httpCompleteOAuth1(a action.Action) error {
	ac := a.Context()
	r := http_action.Request(a)
	up, _ := ac.Upstream.(*Upstream)

	u, status, err := up.completeOAuth1(ac.API, ac.OneTimeStore, r, ac.PluginURL, ac.MattermostUserId)
	if err != nil {
		a.RespondError(status, err)
	}

	return http_action.RespondTemplate(a, "text/html", struct {
		MattermostDisplayName string
		JiraDisplayName       string
		RevokeURL             string
	}{
		JiraDisplayName:       u.UpstreamDisplayName(),
		MattermostDisplayName: u.MattermostDisplayName(),
		RevokeURL:             path.Join(ac.PluginURLPath, jira.RouteUserDisconnect),
	})
}

func httpGetOAuth1PublicKey(a action.Action) error {
	ac := a.Context()
	serverUp := ac.Upstream.(*Upstream)
	pkey, err := serverUp.PublicKeyString()
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err, "failed to load public key")
	}
	return a.RespondPrintf(string(pkey))
}
