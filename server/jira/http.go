// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package jira

import (
	"github.com/mattermost/mattermost-plugin-jira/server/action"
)

var HTTPRoutes = action.AppendRoutes(
	issueHTTPRoutes,
	subscribeHTTPRoutes,
	userHTTPRoutes,
	webhookHTTPRoutes,
)
