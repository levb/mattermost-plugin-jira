// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package action

import (
	"crypto/rsa"
	"fmt"
	"text/template"

	"github.com/pkg/errors"

	"github.com/andygrunwald/go-jira"
	"github.com/dgrijalva/jwt-go"

	"github.com/mattermost/mattermost-plugin-jira/server/kvstore"
	"github.com/mattermost/mattermost-plugin-jira/server/upstream"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

type Config struct {
	OneTimeStore  kvstore.OneTimeStore
	UpstreamStore upstream.UpstreamStore
	Upstream      upstream.Upstream

	// Set by the plugin
	BotIconURL           string
	BotUserId            string
	BotUserName          string
	EnableJiraUI         bool
	API                  plugin.API
	PluginId             string
	PluginVersion        string
	PluginKey            string
	PluginSiteURL        string
	PluginTemplates      map[string]*template.Template
	PluginURL            string
	PluginURLPath        string
	ProxyAuthTokenSecret []byte
	ProxyRSAPrivateKey   *rsa.PrivateKey
	PluginWebhookSecret  string
}

type Context struct {
	Config
	PluginContext *plugin.Context

	MattermostUser      *model.User
	MattermostUserId    string
	MattermostTeamId    string
	MattermostChannelId string

	UpstreamJWT            *jwt.Token
	UpstreamJWTAccountId   string
	UpstreamJWTDisplayName string
	UpstreamJWTUserKey     string
	UpstreamJWTUsername    string
	UpstreamRawJWT         string
	UpstreamUser           upstream.User

	// TODO proxy via an `upstream.Client`?
	JiraClient *jira.Client

	LogError error
}

type Action interface {
	Context() *Context
	FormValue(string) string
	Responder
	Logger
}

type Responder interface {
	RespondTemplate(templateKey, contentType string, values interface{}) error
	RespondJSON(value interface{}) error
	RespondRedirect(redirectURL string) error
	RespondError(httpStatusCode int, err error, wrap ...interface{}) error
	RespondPrintf(format string, args ...interface{}) error
}

type Logger interface {
	Debugf(f string, args ...interface{})
	Infof(f string, args ...interface{})
	Errorf(f string, args ...interface{})
}

type Func func(a Action) error

type actionHandler struct {
	run      Func
	metadata interface{}
}

type action struct {
	context Context
}

var _ Action = (*action)(nil)

func NewAction(config Config) Action {
	return &action{
		context: Context{
			Config: config,
		},
	}
}

func (a *action) Context() *Context {
	return &a.context
}

func (a action) Debugf(f string, args ...interface{}) {
	a.context.API.LogDebug(fmt.Sprintf(f, args...))
}

func (a action) Infof(f string, args ...interface{}) {
	a.context.API.LogInfo(fmt.Sprintf(f, args...))
}

func (a action) Errorf(f string, args ...interface{}) {
	a.context.API.LogError(fmt.Sprintf(f, args...))
}

func (a action) RespondTemplate(templateKey, contentType string, values interface{}) error {
	return nil
}

func (a action) RespondJSON(value interface{}) error {
	return nil
}

func (a action) RespondRedirect(redirectURL string) error {
	return nil
}

func (a action) RespondError(httpStatusCode int, err error, wrap ...interface{}) error {
	return nil
}

func (a action) RespondPrintf(format string, args ...interface{}) error {
	return errors.New("not implemented")
}

func (a action) FormValue(string) string {
	return ""
}
