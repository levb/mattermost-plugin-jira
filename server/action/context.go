// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package action

import (
	"github.com/andygrunwald/go-jira"
	"github.com/dgrijalva/jwt-go"

	"github.com/mattermost/mattermost-plugin-jira/server/config"
	"github.com/mattermost/mattermost-plugin-jira/server/instance"
	"github.com/mattermost/mattermost-plugin-jira/server/store"
	"github.com/mattermost/mattermost-server/model"
	mmplugin "github.com/mattermost/mattermost-server/plugin"
)

type Context struct {
	config.Config

	PluginContext *mmplugin.Context
	Instance         instance.Instance
	User             *store.User
	JiraClient       *jira.Client
	LogErr           error
	MattermostUser   *model.User
	MattermostUserId string
	BackendJWT       *jwt.Token
	BackendRawJWT    string
}
