// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package action

import (
	"fmt"

	"github.com/mattermost/mattermost-plugin-jira/server/config"
	mmplugin "github.com/mattermost/mattermost-server/plugin"
)

type Runner interface {
	Run(a Action) error
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

type Func func(a Action) error

type actionHandler struct {
	run      Func
	metadata interface{}
}

type BasicAction struct {
	context *Context
}

func NewBasicAction(router *Router, conf config.Config, pc *mmplugin.Context, mattermostUserId string) *BasicAction {
	return &BasicAction{
		context: &Context{
			Config:           conf,
			PluginContext:    pc,
			MattermostUserId: mattermostUserId,
		},
	}
}

type Logger interface {
	Debugf(f string, args ...interface{})
	Infof(f string, args ...interface{})
	Errorf(f string, args ...interface{})
}

var _ Logger = (*BasicAction)(nil)

func (a BasicAction) Debugf(f string, args ...interface{}) {
	a.context.API.LogDebug(fmt.Sprintf(f, args...))
}

func (a BasicAction) Infof(f string, args ...interface{}) {
	a.context.API.LogInfo(fmt.Sprintf(f, args...))
}

func (a BasicAction) Errorf(f string, args ...interface{}) {
	a.context.API.LogError(fmt.Sprintf(f, args...))
}

func (a BasicAction) Context() *Context {
	return a.context
}
