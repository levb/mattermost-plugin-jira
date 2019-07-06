// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package jira

import (
	"encoding/json"
	"net/http"

	"github.com/mattermost/mattermost-plugin-jira/server/action"
	"github.com/mattermost/mattermost-plugin-jira/server/action/http_action"
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
	ac := a.Context()
	r := http_action.Request(a)
	req := CreateIssueRequest{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		return a.RespondError(http.StatusBadRequest, err, "failed to decode create issue request")
	}
	issue, status, err := createIssue(ac.API, ac.PluginSiteURL, ac.JiraClient,
		ac.Upstream, ac.MattermostUserId, &req)
	if err != nil {
		return a.RespondError(status, err, "failed to create issue")
	}
	return a.RespondJSON(issue)
}

func httpGetCreateIssueMetadata(a action.Action) error {
	md, err := getCreateIssueMetadata(a.Context().JiraClient)
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err,
			"failed to get create issue metadata")
	}
	return a.RespondJSON(md)
}

func httpGetSearchIssues(a action.Action) error {
	jqlString := a.FormValue("jql")
	summary, status, err := getSearchIssues(a.Context().JiraClient, jqlString)
	if err != nil {
		return a.RespondError(status, err,
			"failed to search for issues")
	}
	return a.RespondJSON(summary)
}

func httpAttachCommentToIssue(a action.Action) error {
	ac := a.Context()
	r := http_action.Request(a)
	req := AttachCommentToIssueRequest{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		return a.RespondError(http.StatusBadRequest, err, "failed to decode attach comment to issue request")
	}
	comment, status, err := attachCommentToIssue(ac.API, ac.PluginSiteURL, ac.JiraClient,
		ac.Upstream, ac.MattermostUserId, req, ac.UpstreamUser)
	if err != nil {
		return a.RespondError(status, err, "failed to attach comment to issue")
	}
	return a.RespondJSON(comment)
}
