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
	routeAPIAttachCommentToIssue   = "/api/v2/attach-comment-to-issue"
	routeAPICreateIssue            = "/api/v2/create-issue"
	routeAPIGetCreateIssueMetadata = "/api/v2/get-create-issue-metadata"
	routeAPIGetSearchIssues        = "/api/v2/get-search-issues"
)

var issueHTTPRoutes = map[string]*action.Route{
	routeAPIGetCreateIssueMetadata: action.NewRoute(
		proxy.RequireHTTPMethod(http.MethodGet),
		proxy.RequireMattermostUserId,
		proxy.RequireUpstream,
		proxy.RequireUpstreamUser,
		RequireClient,
		httpGetCreateIssueMetadata,
	),
	routeAPICreateIssue: action.NewRoute(
		proxy.RequireHTTPMethod(http.MethodPost),
		proxy.RequireMattermostUserId,
		proxy.RequireUpstream,
		proxy.RequireUpstreamUser,
		RequireClient,
		httpCreateIssue,
	),
	routeAPIAttachCommentToIssue: action.NewRoute(
		proxy.RequireHTTPMethod(http.MethodPost),
		proxy.RequireMattermostUserId,
		proxy.RequireUpstream,
		proxy.RequireUpstreamUser,
		RequireClient,
		httpAttachCommentToIssue,
	),
	routeAPIGetSearchIssues: action.NewRoute(
		proxy.RequireHTTPMethod(http.MethodGet),
		proxy.RequireMattermostUserId,
		proxy.RequireUpstream,
		proxy.RequireUpstreamUser,
		RequireClient,
		httpGetSearchIssues,
	),
}

func httpCreateIssue(a action.Action) error {
	// TODO
	return nil
}

func httpGetCreateIssueMetadata(a action.Action) error {
	// TODO
	return nil
}

func httpGetSearchIssues(a action.Action) error {
	// TODO
	return nil
}

func httpAttachCommentToIssue(a action.Action) error {
	// TODO
	return nil
}
