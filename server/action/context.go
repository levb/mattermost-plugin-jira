// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package action

import (
	"text/template"

	"github.com/andygrunwald/go-jira"
	"github.com/dgrijalva/jwt-go"

	"github.com/mattermost/mattermost-plugin-jira/server/instance"
	"github.com/mattermost/mattermost-plugin-jira/server/store"
	"github.com/mattermost/mattermost-server/model"
	mmplugin "github.com/mattermost/mattermost-server/plugin"
)

type Config struct {
	API                  mmplugin.API
	EnsuredStore         store.EnsuredStore
	UserStore            store.UserStore
	InstanceStore        instance.InstanceStore
	CurrentInstanceStore instance.CurrentInstanceStore
	OneTimeStore         store.OneTimeStore
	Templates            map[string]*template.Template // TODO text vs html templates

	MattermostSiteURL string
	PluginId          string
	PluginVersion     string
	PluginKey         string
	PluginURL         string
	PluginURLPath     string

	// BotUserID caches the bot user ID (derived from the configured UserName)
	BotUserId   string
	BotUsername string
	BotIconURL  string

	// Known plugin-wide secrets
	WebhookSecret string
}

type Context struct {
	Config
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
