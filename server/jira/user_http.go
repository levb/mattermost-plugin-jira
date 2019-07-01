// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package jira

import (
	"net/http"

	"github.com/mattermost/mattermost-plugin-jira/server/action"
	"github.com/mattermost/mattermost-plugin-jira/server/proxy"
)

const (
	// APIs for the webapp
	routeAPIGetUserInfo = "/api/v2/userinfo"

	// Generic user connect/disconnect endpoints
	RouteUserConnect    = "/user/connect"
	RouteUserDisconnect = "/user/disconnect"
)

var userHTTPRoutes = map[string]*action.Route{
	routeAPIGetUserInfo: action.NewRoute(
		proxy.RequireHTTPMethod(http.MethodGet),
		proxy.RequireMattermostUserId,
		httpGetUserInfo,
	),

	// Generic user connect/disconnect URLs
	RouteUserConnect: action.NewRoute(
		proxy.RequireUpstream,
		proxy.RequireMattermostUserId,
		connectUser,
	),
	RouteUserDisconnect: action.NewRoute(
		proxy.RequireUpstream,
		proxy.RequireMattermostUserId,
		proxy.RequireMattermostUser,
		disconnectUser,
	),
}

func connectUser(a action.Action) error {
	ac := a.Context()
	redirectURL, status, err := getUserConnectURL(ac.PluginURL,
		ac.OneTimeStore, ac.Upstream, ac.MattermostUserId)
	if err != nil {
		return a.RespondError(status, err)
	}
	return a.RespondRedirect(redirectURL)
}

func disconnectUser(a action.Action) error {
	ac := a.Context()
	err := proxy.DeleteUserNotify(ac.API, ac.Upstream, ac.UpstreamUser)
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err)
	}

	// TODO replace with template
	return a.RespondPrintf(`
<!DOCTYPE html>
<html>
       <head>
               <script>
                       // window.close();
               </script>
       </head>
       <body>
               <p>Disconnected from Jira. Please close this page.</p>
       </body>
</html>
`)
}

func httpGetUserInfo(a action.Action) error {
	ac := a.Context()
	resp := getUserInfo(ac.UpstreamStore, ac.MattermostUserId)

	return a.RespondJSON(resp)
}
