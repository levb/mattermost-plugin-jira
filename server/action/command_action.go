// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package action

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-server/model"
	mmplugin "github.com/mattermost/mattermost-server/plugin"
)

// HTTPAction and CommandAction are declared public so that the plugin can access their
// internals in special cases, Action interface is not mature enough.
type CommandAction struct {
	*BasicAction
	*CommandMetadata

	Args            []string
	ArgsMap         map[string]string
	CommandArgs     *model.CommandArgs
	CommandResponse *model.CommandResponse
}

type CommandMetadata struct {
	Handler Func

	// MinTotalArgs and MaxTotalArgs are applied to the total number of
	// whitespace-separated tokens, including the `/jira` and everything after
	// it.
	MinArgc int
	MaxArgc int

	// ArgNames are for the acual arguments of the command, in the order in
	// which they must appear.
	ArgNames []string
}

var _ Action = (*CommandAction)(nil)

var ErrCommandNotFound = errors.New("command not found")

func MakeCommandAction(router *Router,
	pc *mmplugin.Context, ac Config, commandArgs *model.CommandArgs) (string, *CommandAction, error) {

	argv := strings.Fields(commandArgs.Command)
	if len(argv) == 0 || argv[0] != "/jira" {
		// argv[0] must be "/jira"
		return "", nil, errors.New("MakeCommandAction: unreachable code")
	}
	argv = argv[1:]
	n := len(argv)
	key := ""
	var route *Route
	for ; n > 0; n-- {
		key = strings.Join(argv[:n], "/")
		if router.Routes[key] != nil {
			route = router.Routes[key]
			break
		}
	}
	if key == "" {
		return "", nil, ErrCommandNotFound
	}
	argv = argv[n:]

	// var md *CommandMetadata
	// if route.Metadata != nil
	md, ok := route.Metadata.(*CommandMetadata)
	if !ok || md == nil {
		return "", nil, errors.New("MakeCommandAction: misconfigured router")
	}
	if md.MinArgc >= 0 && len(argv) < md.MinArgc {
		return "", nil, errors.Errorf("expected at least %v arguments", md.MinArgc)
	}
	if md.MaxArgc >= 0 && len(argv) > md.MaxArgc {
		return "", nil, errors.Errorf("expected at most %v arguments", md.MaxArgc)
	}

	// Initialize the FormValue map
	argsMap := map[string]string{}
	for i, arg := range argv {
		if i < len(md.ArgNames) {
			argsMap[md.ArgNames[i]] = arg
		}
		argsMap[fmt.Sprintf("$%v", i+1)] = arg
	}

	a := &CommandAction{
		BasicAction:     NewBasicAction(router, ac, pc),
		CommandMetadata: md,
		Args:            argv,
		ArgsMap:         argsMap,
		CommandArgs:     commandArgs,
		CommandResponse: &model.CommandResponse{},
	}
	a.context.MattermostUserId = commandArgs.UserId
	return key, a, nil
}

func (commandAction *CommandAction) FormValue(name string) string {
	if len(commandAction.ArgsMap) == 0 {
		return ""
	}
	return commandAction.ArgsMap[name]
}

func (commandAction *CommandAction) RespondError(code int, err error, wrap ...interface{}) error {
	if len(wrap) > 0 {
		fmt := wrap[0].(string)
		if err != nil {
			err = errors.WithMessagef(err, fmt, wrap[1:]...)
		} else {
			err = errors.Errorf(fmt, wrap[1:]...)
		}
	}

	if err != nil {
		commandAction.respond(err.Error())
	}
	return err
}

func (commandAction *CommandAction) RespondPrintf(format string, args ...interface{}) error {
	commandAction.respond(fmt.Sprintf(format, args...))
	return nil
}

func (commandAction *CommandAction) RespondRedirect(redirectURL string) error {
	commandAction.CommandResponse = &model.CommandResponse{
		GotoLocation: redirectURL,
	}
	return nil
}

func (commandAction *CommandAction) RespondTemplate(templateKey, contentType string, values interface{}) error {
	t := commandAction.Context().Templates[templateKey]
	if t == nil {
		return commandAction.RespondError(http.StatusInternalServerError, nil,
			"no template found for %q", templateKey)
	}
	bb := &bytes.Buffer{}
	err := t.Execute(bb, values)
	if err != nil {
		return commandAction.RespondError(http.StatusInternalServerError, err,
			"failed to write response")
	}
	commandAction.respond(string(bb.Bytes()))
	return nil
}

func (commandAction *CommandAction) RespondJSON(value interface{}) error {
	bb, err := json.Marshal(value)
	if err != nil {
		return commandAction.RespondError(http.StatusInternalServerError, err,
			"failed to write response")
	}
	commandAction.respond(string(bb))
	return nil
}

func (commandAction *CommandAction) respond(text string) {
	commandAction.CommandResponse = &model.CommandResponse{
		ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
		Text:         text,
		Username:     commandAction.Context().BotUsername,
		IconURL:      commandAction.Context().BotIconURL,
		Type:         model.POST_DEFAULT,
	}
}
