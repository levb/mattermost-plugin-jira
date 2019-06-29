// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package http

import (
	"net/http"

	"github.com/mattermost/mattermost-plugin-jira/server/action"
	"github.com/mattermost/mattermost-plugin-jira/server/action/http_action"
	"github.com/mattermost/mattermost-plugin-jira/server/jira"
	"github.com/mattermost/mattermost-plugin-jira/server/jira/jira_cloud"
	"github.com/mattermost/mattermost-plugin-jira/server/jira/jira_server"
	"github.com/mattermost/mattermost-plugin-jira/server/lib"
	// "github.com/mattermost/mattermost-plugin-jira/server/jira/jira_cloud"
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
	routeWebhookJira           = jira.RouteIncomingWebhook
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
	Default: func(a action.Action) error {
		return a.RespondError(http.StatusNotFound, nil, "not found")
	},
	After: action.Script{http_action.LogAction},
	Routes: map[string]*action.Route{
		// APIs
		routeAPIGetCreateIssueMetadata: action.NewRoute(
			lib.RequireHTTPGet,
			lib.RequireMattermostUserId,
			lib.RequireUpstream,
			lib.RequireUpstreamUser,
			lib.RequireUpstreamClient,
			getCreateIssueMetadata,
		),
		routeAPICreateIssue: action.NewRoute(
			lib.RequireHTTPPost,
			lib.RequireMattermostUserId,
			lib.RequireUpstream,
			lib.RequireUpstreamUser,
			lib.RequireUpstreamClient,
			createIssue,
		),
		routeAPIAttachCommentToIssue: action.NewRoute(
			lib.RequireHTTPPost,
			lib.RequireMattermostUserId,
			lib.RequireUpstream,
			lib.RequireUpstreamUser,
			lib.RequireUpstreamClient,
			attachCommentToIssue,
		),
		routeAPIGetSearchIssues: action.NewRoute(
			lib.RequireHTTPGet,
			lib.RequireMattermostUserId,
			lib.RequireUpstream,
			lib.RequireUpstreamUser,
			lib.RequireUpstreamClient,
			getSearchIssues,
		),
		routeAPIUser: action.NewRoute(
			lib.RequireHTTPGet,
			lib.RequireMattermostUserId,
			getUserInfo,
		),

		// httpChannelSubscriptions already ends in a '/', so adding "*" will
		// pass all sub-paths up to the handler
		routeAPIChannelSubscriptions + "*": action.NewRoute(
			lib.RequireMattermostUserId,
			handleChannelSubscriptions,
		),

		// Incoming webhooks
		routeWebhookJiraSubscribe: action.NewRoute(
			lib.RequireHTTPPost,
			lib.RequireUpstream,
			processSubscribeWebhook,
		),
		routeWebhookJira: action.NewRoute(
			lib.RequireHTTPPost,
			lib.RequireUpstream,
			processLegacyWebhook,
		),
		routeWebhookJiraIssueEvent: action.NewRoute(
			lib.RequireHTTPPost,
			lib.RequireUpstream,
			processLegacyWebhook,
		),

		// Atlassian Connect application
		routeJiraCloudInstalled: action.NewRoute(
			lib.RequireHTTPPost,
			processJiraCloudInstalled,
		),
		routeJiraCloudInstallJSON: action.NewRoute(
			lib.RequireHTTPGet,
			getJiraCloudInstallJSON,
		),

		// User connect and disconnect URLs
		routeUserConnect: action.NewRoute(
			lib.RequireUpstream,
			lib.RequireMattermostUserId,
			connectUser,
		),
		routeUserDisconnect: action.NewRoute(
			lib.RequireUpstream,
			lib.RequireMattermostUserId,
			lib.RequireMattermostUser,
			disconnectUser,
		),

		// Atlassian Connect user mapping
		routeJiraCloudUser: action.NewRoute(
			// TODO this is wrong, all 3 are gets, 2 should be posts
			lib.RequireHTTPGet,
			lib.RequireUpstream,
			jira_cloud.RequireJWT,
			lib.RequireMattermostUserId,
			lib.RequireMattermostUser,
			connectJiraCloudUser),

		// Oauth1 (Jira Server) user mapping
		routeJiraServerOAuth1Complete: action.NewRoute(
			lib.RequireHTTPGet,
			lib.RequireMattermostUserId,
			lib.RequireMattermostUser,
			lib.RequireUpstream,
			completeJiraServerOAuth1),
	},
}
