// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package http

import (
	"net/http"

	"github.com/mattermost/mattermost-plugin-jira/server/jira"

	"github.com/mattermost/mattermost-plugin-jira/server/action"
	"github.com/mattermost/mattermost-plugin-jira/server/filters"
	"github.com/mattermost/mattermost-plugin-jira/server/jira/jiraserver"
)

const (
	// APIs for the webapp
	routeAPIAttachCommentToIssue   = "/api/v2/attach-comment-to-issue"
	routeAPIChannelSubscriptions   = "/api/v2/subscriptions/channel/" // trailing '/' on purpose
	routeAPICreateIssue            = "/api/v2/create-issue"
	routeAPIGetCreateIssueMetadata = "/api/v2/get-create-issue-metadata"
	routeAPIGetSearchIssues        = "/api/v2/get-search-issues"
	routeAPIUser                   = "/api/v2/userinfo"
	routeAPIUserSettings           = "/api/v2/settingsinfo"

	// Generic user connect/disconnect endpoints
	routeUserConnect    = "/user/connect"
	routeUserDisconnect = "/user/disconnect"

	// Jira incoming webhooks
	routeWebhookJira           = app.RouteIncomingWebhook
	routeWebhookJiraIssueEvent = "/issue_event"
	routeWebhookJiraSubscribe  = "/api/v2/webhook"

	// Jira Cloud specific routes
	routeJiraCloudInstalled        = "/ac/installed"
	routeJiraCloudInstallJSON      = "/ac/atlassian-connect.json"
	routeJiraCloudUninstalled      = "/ac/uninstalled"
	routeJiraCloudUser             = "/ac/*"
	routeJiraCloudUserConfirm      = "/ac/user_confirm.html"
	routeJiraCloudUserConnected    = "/ac/user_connected.html"
	routeJiraCloudUserDisconnected = "/ac/user_disconnected.html"

	// Jira Server specific routes
	routeJiraServerOAuth1Complete  = jira_server.RouteOAuth1Complete
	routeJiraServerOAuth1PublicKey = "/oauth1/public_key.html"
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
		routeAPIUser: action.NewRoute(
			filters.RequireHTTPGet,
			filters.RequireMattermostUserId,
			getUserInfo,
		),

		// httpChannelSubscriptions already ends in a '/', so adding "*" will
		// pass all sub-paths up to the handler
		routeAPIChannelSubscriptions + "*": action.NewRoute(
			filters.RequireMattermostUserId,
			handleChannelSubscriptions,
		),

		// Incoming webhooks
		routeWebhookJiraSubscribe: action.NewRoute(
			filters.RequireHTTPPost,
			filters.RequireInstance,
			processSubscribeWebhook,
		),
		routeWebhookJira: action.NewRoute(
			filters.RequireHTTPPost,
			filters.RequireInstance,
			processLegacyWebhook,
		),
		routeWebhookJiraIssueEvent: action.NewRoute(
			filters.RequireHTTPPost,
			filters.RequireInstance,
			processLegacyWebhook,
		),

		// Atlassian Connect application
		routeJiraCloudInstalled: action.NewRoute(
			filters.RequireHTTPPost,
			processJiraCloudInstalled,
		),
		routeJiraCloudInstallJSON: action.NewRoute(
			filters.RequireHTTPGet,
			getJiraCloudInstallJSON,
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
		routeJiraCloudUser: action.NewRoute(
			// TODO this is wrong, all 3 are gets, 2 should be posts
			filters.RequireHTTPGet,
			filters.RequireInstance,
			filters.RequireJiraCloudJWT,
			filters.RequireMattermostUserId,
			filters.RequireMattermostUser,
			connectJiraCloudUser),

		// Oauth1 (Jira Server) user mapping
		routeJiraServerOAuth1Complete: action.NewRoute(
			filters.RequireHTTPGet,
			filters.RequireMattermostUserId,
			filters.RequireMattermostUser,
			filters.RequireInstance,
			completeJiraServerOAuth1),
	},
}
