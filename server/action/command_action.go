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

	"github.com/mattermost/mattermost-plugin-jira/server/config"
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

// MakeCommandAction makes a new CommandAction. In case of an error, it still
// returns a non-nil CommandAction so that the caller can RespondXXX as needed
func MakeCommandAction(router *Router,
	pc *mmplugin.Context, conf config.Config, commandArgs *model.CommandArgs) (string, *CommandAction, error) {

	a := &CommandAction{
		BasicAction:     NewBasicAction(router, conf, pc, commandArgs.UserId),
		CommandArgs:     commandArgs,
		CommandResponse: &model.CommandResponse{},
	}

	argv := strings.Fields(commandArgs.Command)
	if len(argv) == 0 || argv[0] != "/jira" {
		// argv[0] must be "/jira"
		return "", a, errors.New("MakeCommandAction: unreachable code")
	}
	n := len(argv)
	key := ""
	var route *Route
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
	a.Args = argv

	md := &CommandMetadata{}
	if route.Metadata != nil {
		md, _ = route.Metadata.(*CommandMetadata)
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
	a.CommandMetadata = md

	// Initialize the FormValue map
	argsMap := map[string]string{}
	for i, arg := range argv {
		if i < len(md.ArgNames) {
			argsMap[md.ArgNames[i]] = arg
		}
		argsMap[fmt.Sprintf("$%v", i+1)] = arg
	}
	a.ArgsMap = argsMap

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
		Username:     commandAction.Context().UserName,
		IconURL:      commandAction.Context().BotIconURL,
		Type:         model.POST_DEFAULT,
	}
}
