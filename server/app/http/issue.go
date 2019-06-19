// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package http

import (
	"github.com/mattermost/mattermost-plugin-jira/server/action"
	"github.com/mattermost/mattermost-plugin-jira/server/app"
)

func createIssue(a action.Action) error {
	err := action.Script{
		app.RequireHTTPPost,
		app.RequireMattermostUserId,
		app.RequireInstance,
		app.RequireBackendUser,
		app.RequireJiraClient,
	}.Run(a)
	if err != nil {
		return err
	}

	return nil
}

func getCreateIssueMetadata(a action.Action) error {
	err := action.Script{
		app.RequireHTTPGet,
		app.RequireMattermostUserId,
		app.RequireInstance,
		app.RequireBackendUser,
		app.RequireJiraClient,
	}.Run(a)
	if err != nil {
		return err
	}
	return nil
}

func getSearchIssues(a action.Action) error {
	err := action.Script{
		app.RequireHTTPGet,
		app.RequireMattermostUserId,
		app.RequireInstance,
		app.RequireBackendUser,
		app.RequireJiraClient,
	}.Run(a)
	if err != nil {
		return err
	}
	return nil
}

func attachCommentToIssue(a action.Action) error {
	err := action.Script{
		app.RequireHTTPPost,
		app.RequireMattermostUserId,
		app.RequireInstance,
		app.RequireBackendUser,
		app.RequireJiraClient,
	}.Run(a)
	if err != nil {
		return err
	}
	return nil
}
