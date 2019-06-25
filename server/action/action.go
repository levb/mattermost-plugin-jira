// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package action

import (
	"fmt"
	"github.com/pkg/errors"

	"github.com/andygrunwald/go-jira"
	"github.com/dgrijalva/jwt-go"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
	"github.com/mattermost/mattermost-plugin-jira/server/config"
	"github.com/mattermost/mattermost-plugin-jira/server/upstream"
)

type Context struct {
	config.Config `json:"none"`

	PluginContext *plugin.Context
	Upstream         upstream.Upstream
	User             upstream.User
	JiraClient       *jira.Client
	LogErr           error
	MattermostUser   *model.User
	MattermostUserId string
	UpstreamJWT       *jwt.Token
	UpstreamRawJWT    string
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

func NewAction(router *Router, conf config.Config, pc *plugin.Context, mattermostUserId string) Action {
	return &action{
		context: Context{
			Config:           conf,
			PluginContext:    pc,
			MattermostUserId: mattermostUserId,
		},
	}
}

func (a action) Context() *Context {
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

func (a action) RespondRedirect(redirectURL string) error{
	return nil
}

func (a action) RespondError(httpStatusCode int, err error, wrap ...interface{})error {
	return nil
}

func (a action) RespondPrintf(format string, args ...interface{}) error {
	return errors.New("not implemented")	
}

func (a action) FormValue(string) string {
	return ""
}