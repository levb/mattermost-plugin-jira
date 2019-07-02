// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package plugin

import (
	"net/http"

	"github.com/mattermost/mattermost-plugin-jira/server/action"
	"github.com/mattermost/mattermost-plugin-jira/server/action/http_action"
	"github.com/mattermost/mattermost-plugin-jira/server/jira"
	"github.com/mattermost/mattermost-plugin-jira/server/jira/jira_cloud"
	"github.com/mattermost/mattermost-plugin-jira/server/jira/jira_server"
)

var httpRouter = &action.Router{
	Before: action.Script{http_action.Require},
	Default: func(a action.Action) error {
		return a.RespondError(http.StatusNotFound, nil, "not found")
	},
	After:  action.Script{http_action.LogAction},
	Routes: map[string]*action.Route{},
}

func init() {
	httpRouter.AddRoutes(jira.HTTPRoutes)
	httpRouter.AddRoutes(jira_cloud.HTTPRoutes)
	httpRouter.AddRoutes(jira_server.HTTPRoutes)
}
