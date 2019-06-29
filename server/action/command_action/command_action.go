// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package command_action

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-jira/server/action"
	"github.com/mattermost/mattermost-server/model"
)

// HTTPAction and CommandAction are declared public so that the plugin can access their
// internals in special cases, Action interface is not mature enough.
type Action struct {
	action.Action
	*Metadata

	args            []string
	argsMap         map[string]string
	commandArgs     *model.CommandArgs
	CommandResponse *model.CommandResponse
}

type Metadata struct {
	// MinTotalArgs and MaxTotalArgs are applied to the total number of
	// whitespace-separated tokens, including the `/jira` and everything after
	// it.
	MinArgc int
	MaxArgc int

	// ArgNames are for the acual arguments of the command, in the order in
	// which they must appear.
	ArgNames []string
}

var _ action.Action = (*Action)(nil)

// Make makes a new command Action. In case of an error, it still
// returns a non-nil Action so that the caller can RespondXXX as needed
func Make(router action.Router, inner action.Action, commandArgs *model.CommandArgs) (string, action.Action, error) {

	a := &Action{
		Action:          inner,
		commandArgs:     commandArgs,
		CommandResponse: &model.CommandResponse{},
	}

	a.Context().MattermostUserId = commandArgs.UserId

	argv := strings.Fields(commandArgs.Command)
	if len(argv) == 0 || argv[0] != "/jira" {
		// argv[0] must be "/jira"
		return "", a, errors.New("MakeCommandAction: unreachable code")
	}
	n := len(argv)
	key := ""
	var route *action.Route
	for ; n > 1; n-- {
		key = strings.Join(argv[1:n], "/")
		if router.Routes[key] != nil {
			route = router.Routes[key]
			break
		}
	}
	if route == nil {
		// execute the default
		return "", a, nil
	}
	commandItself := strings.Join(argv, " ")
	argv = argv[n:]
	a.args = argv

	md := &Metadata{}
	if route.Metadata != nil {
		md, _ = route.Metadata.(*Metadata)
		if md == nil {
			return "", a, errors.Errorf(
				"MakeCommandAction: misconfigured router: wrong CommandMetadata type %T", route.Metadata)
		}
	}

	if md.MinArgc >= 0 && len(argv) < md.MinArgc {
		return "", a, errors.Errorf(
			"expected at least %v arguments after %q", md.MinArgc, commandItself)
	}
	if md.MaxArgc >= 0 && len(argv) > md.MaxArgc {
		return "", a, errors.Errorf(
			"expected at most %v arguments after %q", md.MaxArgc, commandItself)
	}
	a.Metadata = md

	// Initialize the FormValue map
	argsMap := map[string]string{}
	for i, arg := range argv {
		if i < len(md.ArgNames) {
			argsMap[md.ArgNames[i]] = arg
		}
		argsMap[fmt.Sprintf("$%v", i+1)] = arg
	}
	a.argsMap = argsMap

	return key, a, nil
}

func RequireCommandAction(a action.Action) error {
	_, ok := a.(*Action)
	if !ok {
		return errors.Errorf("wrong action type, expected HTTPAction, got %T", a)
	}
	return nil
}

func (a Action) FormValue(name string) string {
	if len(a.argsMap) == 0 {
		return ""
	}
	return a.argsMap[name]
}

func (a *Action) RespondError(code int, err error, wrap ...interface{}) error {
	if len(wrap) > 0 {
		fmt := wrap[0].(string)
		if err != nil {
			err = errors.WithMessagef(err, fmt, wrap[1:]...)
		} else {
			err = errors.Errorf(fmt, wrap[1:]...)
		}
	}

	if err != nil {
		a.respond(err.Error())
	}
	return err
}

func (a *Action) RespondPrintf(format string, args ...interface{}) error {
	a.respond(fmt.Sprintf(format, args...))
	return nil
}

func (a *Action) RespondRedirect(redirectURL string) error {
	a.CommandResponse = &model.CommandResponse{
		GotoLocation: redirectURL,
	}
	return nil
}

func (a *Action) RespondTemplate(templateKey, contentType string, values interface{}) error {
	t := a.Context().Templates[templateKey]
	if t == nil {
		return a.RespondError(http.StatusInternalServerError, nil,
			"no template found for %q", templateKey)
	}
	bb := &bytes.Buffer{}
	err := t.Execute(bb, values)
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err,
			"failed to write response")
	}
	a.respond(string(bb.Bytes()))
	return nil
}

func (a *Action) RespondJSON(value interface{}) error {
	bb, err := json.Marshal(value)
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err,
			"failed to write response")
	}
	a.respond(string(bb))
	return nil
}

func (a *Action) respond(text string) {
	a.CommandResponse = &model.CommandResponse{
		ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
		Text:         text,
		Username:     a.Context().BotUserName,
		IconURL:      a.Context().BotIconURL,
		Type:         model.POST_DEFAULT,
	}
}

func LogAction(a action.Action) error {
	commandAction, ok := a.(*Action)
	ac := a.Context()
	switch {
	case !ok:
		a.Errorf("command: %q error: misconfiguration, wrong Action type", commandAction.commandArgs.Command)
	case ac.LogErr != nil:
		a.Infof("command: %q error:%v", commandAction.commandArgs.Command, ac.LogErr)
	default:
		a.Debugf("command: %q", commandAction.commandArgs.Command)
	}
	return nil
}
