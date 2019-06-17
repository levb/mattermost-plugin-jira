// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package action

import (
	"fmt"
	"net/http"

	"github.com/mattermost/mattermost-server/model"
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

func NewBasicAction(router *Router, cc ConfiguredContext, pc *mmplugin.Context) *BasicAction {
	return &BasicAction{
		context: &Context{
			ConfiguredContext: cc,
			PluginContext:     pc,
		},
	}
}

func RequireMattermostUserId(a Action) error {
	ac := a.Context()
	if ac.MattermostUserId == "" {
		return a.RespondError(http.StatusUnauthorized, nil,
			"not authorized")
	}
	// MattermostUserId is set by the protocol-specific New...Action, nothing to do here
	return nil
}

func RequireMattermostUser(a Action) error {
	ac := a.Context()
	if ac.MattermostUser != nil {
		return nil
	}
	err := Script{RequireMattermostUserId}.Run(a)
	if err != nil {
		return err
	}

	mattermostUser, appErr := ac.API.GetUser(ac.MattermostUserId)
	if appErr != nil {
		return a.RespondError(http.StatusInternalServerError, appErr,
			"failed to load Mattermost user Id:%s", ac.MattermostUserId)
	}
	ac.MattermostUser = mattermostUser
	return nil
}

func RequireMattermostSysAdmin(a Action) error {
	err := Script{RequireMattermostUser}.Run(a)
	if err != nil {
		return err
	}

	ac := a.Context()
	if !ac.MattermostUser.IsInRole(model.SYSTEM_ADMIN_ROLE_ID) {
		return a.RespondError(http.StatusUnauthorized, nil,
			"reserverd for system administrators")
	}
	return nil
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
