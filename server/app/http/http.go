// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package http

import (
	"net/http"

	"github.com/mattermost/mattermost-plugin-jira/server/app"

	"github.com/mattermost/mattermost-plugin-jira/server/action"
	"github.com/mattermost/mattermost-plugin-jira/server/filters"
	"github.com/mattermost/mattermost-plugin-jira/server/instance/jira_server"
)

const (
	routeACInstalled               = "/ac/installed"
	routeACJSON                    = "/ac/atlassian-connect.json"
	routeACUninstalled             = "/ac/uninstalled"
	routeACUser                    = "/ac/*"
	routeACUserConfirm             = "/ac/user_confirm.html"
	routeACUserConnected           = "/ac/user_connected.html"
	routeACUserDisconnected        = "/ac/user_disconnected.html"
	routeAPIAttachCommentToIssue   = "/api/v2/attach-comment-to-issue"
	routeAPICreateIssue            = "/api/v2/create-issue"
	routeAPIGetCreateIssueMetadata = "/api/v2/get-create-issue-metadata"
	routeAPIGetSearchIssues        = "/api/v2/get-search-issues"
	routeAPISubscribeWebhook       = "/api/v2/webhook"
	routeAPISubscriptionsChannel   = "/api/v2/subscriptions/channel/" // trailing '/' on purpose
	routeAPIUserInfo               = "/api/v2/userinfo"
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
		routeAPIGetCreateIssueMetadata: action.NewRoute(
			filters.RequireHTTPGet,
			filters.RequireMattermostUserId,
			filters.RequireInstance,
			filters.RequireBackendUser,
			filters.RequireJiraClient,
			getCreateIssueMetadata,
		),
		routeAPICreateIssue: action.NewRoute(
			filters.RequireHTTPPost,
			filters.RequireMattermostUserId,
			filters.RequireInstance,
			filters.RequireBackendUser,
			filters.RequireJiraClient,
			createIssue,
		),
		routeAPIAttachCommentToIssue: action.NewRoute(
			filters.RequireHTTPPost,
			filters.RequireMattermostUserId,
			filters.RequireInstance,
			filters.RequireBackendUser,
			filters.RequireJiraClient,
			attachCommentToIssue,
		),
		routeAPIGetSearchIssues: action.NewRoute(
			filters.RequireHTTPGet,
			filters.RequireMattermostUserId,
			filters.RequireInstance,
			filters.RequireBackendUser,
			filters.RequireJiraClient,
			getSearchIssues,
		),
		routeAPIUserInfo: action.NewRoute(
			filters.RequireHTTPGet,
			filters.RequireMattermostUserId,
			getUserInfo,
		),
		routeAPISubscribeWebhook: action.NewRoute(
			filters.RequireHTTPPost,
			filters.RequireInstance,
			processSubscribeWebhook,
		),

		// httpChannelSubscriptions already ends in a '/', so adding "*" will
		// pass all sub-paths up to the handler
		routeAPISubscriptionsChannel + "*": action.NewRoute(
			filters.RequireMattermostUserId,
			handleChannelSubscriptions,
		),

		// Incoming webhooks
		routeIncomingWebhook: action.NewRoute(
			filters.RequireHTTPPost,
			filters.RequireInstance,
			processLegacyWebhook,
		),
		routeIncomingIssueEvent: action.NewRoute(
			filters.RequireHTTPPost,
			filters.RequireInstance,
			processLegacyWebhook,
		),

		// Atlassian Connect application
		routeACInstalled: action.NewRoute(
			filters.RequireHTTPPost,
			processACInstalled,
		),
		routeACJSON: action.NewRoute(
			filters.RequireHTTPGet,
			getACJSON,
		),

		// User connect and disconnect URLs
		routeUserConnect: action.NewRoute(
			filters.RequireInstance,
			filters.RequireMattermostUserId,
			connectUser,
		),
		routeUserDisconnect: action.NewRoute(
			filters.RequireInstance,
			filters.RequireMattermostUserId,
			filters.RequireMattermostUser,
			disconnectUser,
		),

		// Atlassian Connect user mapping
		routeACUser: action.NewRoute(
			// TODO this is wrong, all 3 are gets, 2 should be posts
			filters.RequireHTTPGet,
			filters.RequireInstance,
			filters.RequireJiraCloudJWT,
			filters.RequireMattermostUserId,
			filters.RequireMattermostUser,
			connectAC),

		// Oauth1 (Jira Server) user mapping
		routeOAuth1Complete: action.NewRoute(
			filters.RequireHTTPGet,
			filters.RequireMattermostUserId,
			filters.RequireMattermostUser,
			filters.RequireInstance,
			completeOAuth1),
	},
}
