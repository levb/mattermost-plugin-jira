// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package jira_cloud

import (
	"net/http"
	"path"

	"github.com/mattermost/mattermost-plugin-jira/server/action"
	"github.com/mattermost/mattermost-plugin-jira/server/action/http_action"
	"github.com/mattermost/mattermost-plugin-jira/server/jira"
	"github.com/mattermost/mattermost-plugin-jira/server/proxy"
	"github.com/mattermost/mattermost-plugin-jira/server/upstream"
	"github.com/mattermost/mattermost-server/model"
)

const (
	routeInstalled       = "/ac/installed"
	routeInstallJSON     = "/ac/atlassian-connect.json"
	routeUninstalled     = "/ac/uninstalled"
	routeConnectRedirect = "/ac/user_redirect.html"
	routeConnectConfirm  = "/ac/user_confirm.html"
	routeConnect         = "/ac/user_connected.html"
	routeDisconnect      = "/ac/user_disconnected.html"
)

const (
	argJiraJWT = "jwt"
	argMMToken = "mm_token"
)

var HTTPRoutes = map[string]*action.Route{
	routeInstallJSON: action.NewRoute(
		proxy.RequireHTTPMethod(http.MethodGet),
		httpInstallJSON,
	),
	routeInstalled: action.NewRoute(
		proxy.RequireHTTPMethod(http.MethodPost),
		httpInstalled,
	),
	routeConnectRedirect: action.NewRoute(
		proxy.RequireHTTPMethod(http.MethodGet),
		proxy.RequireMattermostUserId,
		proxy.RequireUpstream,
		RequireUpstream,
		RequireJWT,
		httpConnectRedirect,
	),
	routeConnectConfirm: action.NewRoute(
		proxy.RequireHTTPMethod(http.MethodGet),
		proxy.RequireMattermostUserId,
		proxy.RequireMattermostUser,
		proxy.RequireUpstream,
		RequireUpstream,
		RequireJWT,
		httpConnectConfirm,
	),
	routeConnect: action.NewRoute(
		proxy.RequireHTTPMethod(http.MethodGet),
		proxy.RequireMattermostUserId,
		proxy.RequireMattermostUser,
		proxy.RequireUpstream,
		RequireUpstream,
		RequireJWT,
		httpConnect,
	),
	routeDisconnect: action.NewRoute(
		proxy.RequireHTTPMethod(http.MethodGet),
		proxy.RequireMattermostUserId,
		proxy.RequireMattermostUser,
		proxy.RequireUpstream,
		proxy.RequireUpstreamUser,
		RequireUpstream,
		RequireJWT,
		httpDisconnect,
	),
}

func httpInstallJSON(a action.Action) error {
	ac := a.Context()
	return http_action.RespondTemplate(a,
		"application/json",
		map[string]string{
			"BaseURL":             ac.PluginURL,
			"RouteACJSON":         routeInstallJSON,
			"RouteACInstalled":    routeInstalled,
			"RouteACUninstalled":  routeUninstalled,
			"RouteACUserRedirect": routeConnectRedirect,
			"UserLandingPageKey":  userLandingPageKey,
			"ExternalURL":         ac.MattermostSiteURL,
			"PluginKey":           ac.PluginKey,
		})
}

func httpInstalled(a action.Action) error {
	ac := a.Context()
	r := http_action.Request(a)

	status, err := processInstalled(ac.API, ac.UpstreamStore, ac.OneTimeStore,
		ac.AuthTokenSecret, r.Body)
	if err != nil {
		return a.RespondError(status, err,
			"failed to process atlassian-connect installed event")
	}

	return a.RespondJSON([]string{"OK"})
}

func httpConnectRedirect(a action.Action) error {
	ac := a.Context()
	request := http_action.Request(a)

	return a.RespondTemplate(request.URL.Path, "text/html", struct {
		RedirectURL string
		ArgJiraJWT  string
		ArgMMToken  string
	}{
		ArgJiraJWT:  argJiraJWT,
		ArgMMToken:  argMMToken,
		RedirectURL: path.Join(ac.PluginURLPath, routeConnectConfirm),
	})
}

func httpConnectConfirm(a action.Action) error {
	u, _, err := userFromTokens(a)
	if err != nil {
		return err
	}
	return respondConnectTemplate(a, u)
}

func httpConnect(a action.Action) error {
	ac := a.Context()
	u, secret, err := userFromTokens(a)
	if err != nil {
		return err
	}
	status, err := processUserConnected(ac.API, ac.Upstream, ac.OneTimeStore,
		u, secret, ac.MattermostUserId)
	if err != nil {
		a.RespondError(status, err)
	}
	a.Debugf("Stored and notified: %s %+v", ac.MattermostUserId, ac.UpstreamUser)
	return respondConnectTemplate(a, u)
}

func httpDisconnect(a action.Action) error {
	ac := a.Context()

	u, _, err := userFromTokens(a)
	if err != nil {
		return err
	}
	status, err := processUserDisconnected(ac.API, ac.Upstream, ac.UpstreamUser)
	if err != nil {
		a.RespondError(status, err)
	}
	a.Debugf("Deleted and notified: %s %+v", ac.MattermostUserId, ac.UpstreamUser)

	return respondConnectTemplate(a, u)
}

func respondConnectTemplate(a action.Action, u *jira.User) error {
	ac := a.Context()
	mmtoken := a.FormValue(argMMToken)
	r := http_action.Request(a)
	// This set of props should work for all relevant routes/templates
	return a.RespondTemplate(r.URL.Path, "text/html", struct {
		ConnectSubmitURL      string
		DisconnectSubmitURL   string
		ArgJiraJWT            string
		ArgMMToken            string
		MMToken               string
		JiraDisplayName       string
		MattermostDisplayName string
	}{
		DisconnectSubmitURL:   path.Join(ac.PluginURLPath, routeDisconnect),
		ConnectSubmitURL:      path.Join(ac.PluginURLPath, routeConnect),
		ArgJiraJWT:            argJiraJWT,
		ArgMMToken:            argMMToken,
		MMToken:               mmtoken,
		JiraDisplayName:       u.UpstreamDisplayName(),
		MattermostDisplayName: ac.MattermostUser.GetDisplayName(model.SHOW_NICKNAME_FULLNAME),
	})
}

func userFromTokens(a action.Action) (*jira.User, string, error) {
	ac := a.Context()
	mmtoken := a.FormValue(argMMToken)
	up := ac.Upstream.(*Upstream)

	juser := jira.JiraUser{
		AccountID:   ac.UpstreamJWTAccountId,
		DisplayName: ac.UpstreamJWTDisplayName,
		Key:         ac.UpstreamJWTUserKey,
		Name:        ac.UpstreamJWTUsername,
	}
	requestedUserId, secret, err := up.parseAuthToken(mmtoken)
	if err != nil {
		return nil, "", a.RespondError(http.StatusUnauthorized, err)
	}

	if ac.MattermostUserId != requestedUserId {
		return nil, "", a.RespondError(http.StatusUnauthorized, nil, "not authorized, user mismatch")
	}

	return &jira.User{
		BasicUser: *upstream.NewBasicUser(ac.MattermostUserId, ac.UpstreamJWTAccountId),
		JiraUser:  juser,
	}, secret, nil
}
