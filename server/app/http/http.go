// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package http

import (
	"net/http"

	"github.com/mattermost/mattermost-plugin-jira/server/app"

	"github.com/mattermost/mattermost-plugin-jira/server/action"
	"github.com/mattermost/mattermost-plugin-jira/server/instance/jira_server"
)

const (
	routeAPICreateIssue            = "/api/v2/create-issue"
	routeAPIGetCreateIssueMetadata = "/api/v2/get-create-issue-metadata"
	routeAPIGetSearchIssues        = "/api/v2/get-search-issues"
	routeAPIAttachCommentToIssue   = "/api/v2/attach-comment-to-issue"
	routeAPIUserInfo               = "/api/v2/userinfo"
	routeAPISubscribeWebhook       = "/api/v2/webhook"
	routeAPISubscriptionsChannel   = "/api/v2/subscriptions/channel/" // trailing '/' on purpose
	routeACInstalled               = "/ac/installed"
	routeACJSON                    = "/ac/atlassian-connect.json"
	routeACUninstalled             = "/ac/uninstalled"
	routeACUserRedirectWithToken   = "/ac/user_redirect.html"
	routeACUserConfirm             = "/ac/user_confirm.html"
	routeACUserConnected           = "/ac/user_connected.html"
	routeACUserDisconnected        = "/ac/user_disconnected.html"
	routeIncomingIssueEvent        = "/issue_event"
	routeIncomingWebhook           = app.RouteIncomingWebhook
	routeOAuth1Complete            = jira_server.RouteOAuth1Complete
	routeOAuth1PublicKey           = "/oauth1/public_key.html"
	routeUserConnect               = "/user/connect"
	routeUserDisconnect            = "/user/disconnect"
)

var Router = &action.Router{
	DefaultHandler: func(a action.Action) error {
		return a.RespondError(http.StatusNotFound, nil, "not found")
	},
	LogHandler: action.HTTPLogHandler,
	Routes: map[string]*action.Route{
		// APIs
		routeAPICreateIssue:            action.NewHTTPRoute(createIssue),
		routeAPIAttachCommentToIssue:   action.NewHTTPRoute(attachCommentToIssue),
		routeAPIGetSearchIssues:        action.NewHTTPRoute(getSearchIssues),
		routeAPIGetCreateIssueMetadata: action.NewHTTPRoute(getCreateIssueMetadata),
		routeAPIUserInfo:               action.NewHTTPRoute(getUserInfo),
		routeAPISubscribeWebhook:       action.NewHTTPRoute(processSubscribeWebhook),

		// httpChannelSubscriptions already ends in a '/', so adding "*" will
		// pass all sub-paths up to the handler
		routeAPISubscriptionsChannel + "*": action.NewHTTPRoute(handleChannelSubscriptions),

		// Incoming webhooks
		routeIncomingWebhook:    action.NewHTTPRoute(processLegacyWebhook),
		routeIncomingIssueEvent: action.NewHTTPRoute(processLegacyWebhook),

		// Atlassian Connect application
		routeACInstalled: action.NewHTTPRoute(processACInstalled),
		routeACJSON:      action.NewHTTPRoute(getACJSON),

		// User connect and disconnect URLs
		// routeUserConnect:    httpUserConnect,
		// routeUserDisconnect: httpUserDisconnect,

		// Atlassian Connect user mapping
		// routeACUserRedirectWithToken: httpACUserRedirect,
		// routeACUserConfirm:           httpACUserConfirm,
		// routeACUserConnected:         httpACUserConnected,
		// routeACUserDisconnected:      httpACUserDisconnected,

		// // Oauth1 (Jira Server) user mapping
		// routeOAuth1Complete: httpOAuth1Complete,
	},
}
