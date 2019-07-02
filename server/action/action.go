// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package action

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/andygrunwald/go-jira"
	"github.com/dgrijalva/jwt-go"

	"github.com/mattermost/mattermost-plugin-jira/server/context"
	"github.com/mattermost/mattermost-plugin-jira/server/upstream"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

// Context has pre-computed values for an action
type Context struct {
	*context.Context `json:"-"`

	PluginContext *plugin.Context
	LogErr        error

	MattermostUser   *model.User
	MattermostUserId string

	Upstream               upstream.Upstream
	UpstreamJWT            *jwt.Token
	UpstreamJWTDisplayName string
	UpstreamJWTUserKey     string
	UpstreamJWTUsername    string
	UpstreamRawJWT         string
	UpstreamUser           upstream.User

	// TODO proxy via an `upstream.Client`?
	JiraClient *jira.Client
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
	context *Context
}

var _ Action = (*action)(nil)

func NewAction(router *Router, context context.Context, pc *plugin.Context, mattermostUserId string) Action {
	return &action{
		context: &Context{
			Context:          &context,
			PluginContext:    pc,
			MattermostUserId: mattermostUserId,
		},
	}
}

func (a *action) Context() *Context {
	return a.context
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
