// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package action

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mattermost/mattermost-plugin-jira/server/config"
	mmplugin "github.com/mattermost/mattermost-server/plugin"
	"github.com/pkg/errors"
)

type HTTPAction struct {
	*BasicAction

	Request            *http.Request
	ResponseWriter     http.ResponseWriter
	ResponseStatusCode int
}

var _ Action = (*HTTPAction)(nil)

func MakeHTTPAction(router *Router, pc *mmplugin.Context, conf config.Config, r *http.Request, w http.ResponseWriter) *HTTPAction {
	mattermostUserId := r.Header.Get("Mattermost-User-Id")
	a := &HTTPAction{
		BasicAction:    NewBasicAction(router, conf, pc, mattermostUserId),
		Request:        r,
		ResponseWriter: w,
	}
	return a
}

func (httpAction HTTPAction) FormValue(key string) string {
	return httpAction.Request.FormValue(key)
}

func (httpAction HTTPAction) RespondError(code int, err error, wrap ...interface{}) error {
	if len(wrap) > 0 {
		fmt := wrap[0].(string)
		if err != nil {
			err = errors.WithMessagef(err, fmt, wrap[1:]...)
		} else {
			err = errors.Errorf(fmt, wrap[1:]...)
		}
	}

	if err == nil {
		return nil
	}

	httpAction.ResponseStatusCode = code
	http.Error(httpAction.ResponseWriter, err.Error(), code)
	return err
}

func (httpAction HTTPAction) RespondPrintf(format string, args ...interface{}) error {
	text := fmt.Sprintf(format, args...)
	httpAction.ResponseWriter.Header().Set("Content-Type", "text/plain")
	_, err := httpAction.ResponseWriter.Write([]byte(text))
	if err != nil {
		return httpAction.RespondError(http.StatusInternalServerError, err,
			"failed to write response")
	}
	return nil
}

func (httpAction HTTPAction) RespondRedirect(redirectURL string) error {
	status := http.StatusFound
	if httpAction.Request.Method != http.MethodGet {
		status = http.StatusTemporaryRedirect
	}
	http.Redirect(httpAction.ResponseWriter, httpAction.Request, redirectURL, status)
	httpAction.ResponseStatusCode = status
	return nil
}

func (httpAction HTTPAction) RespondTemplate(templateKey, contentType string, values interface{}) error {
	t := httpAction.context.Templates[templateKey]
	if t == nil {
		return httpAction.RespondError(http.StatusInternalServerError, nil,
			"no template found for %q", templateKey)
	}
	httpAction.ResponseWriter.Header().Set("Content-Type", contentType)
	err := t.Execute(httpAction.ResponseWriter, values)
	if err != nil {
		return httpAction.RespondError(http.StatusInternalServerError, err,
			"failed to write response")
	}
	return nil
}

func (httpAction HTTPAction) RespondJSON(value interface{}) error {
	httpAction.ResponseWriter.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(httpAction.ResponseWriter).Encode(value)
	if err != nil {
		return httpAction.RespondError(http.StatusInternalServerError, err,
			"failed to write response")
	}
	return nil
}

func HTTPLogHandler(a Action) error {
	httpAction, ok := a.(*HTTPAction)
	if !ok {
		return a.RespondError(http.StatusInternalServerError, nil,
			"Wrong action type %T, eexpected HTTPAction", a)
	}
	if httpAction.ResponseStatusCode == 0 {
		httpAction.ResponseStatusCode = http.StatusOK
	}
	if httpAction.context.LogErr != nil {
		a.Infof("http: %v %s %v", httpAction.ResponseStatusCode, httpAction.Request.URL.String(),
			httpAction.context.LogErr)
	} else {
		a.Debugf("http: %v %s", httpAction.ResponseStatusCode, httpAction.Request.URL.String())
	}
	return nil
}

func HTTPRespondTemplate(a Action, contentType string, values interface{}) error {
	httpAction, ok := a.(HTTPAction)
	if !ok {
		return a.RespondError(http.StatusInternalServerError, nil,
			"Wrong action type %T, expected HTTPAction", a)
	}
	return a.RespondTemplate(httpAction.Request.URL.Path, contentType, values)
}

func HTTPRequest(a Action) (*http.Request, error) {
	httpAction, ok := a.(HTTPAction)
	if !ok {
		return nil, a.RespondError(http.StatusInternalServerError, nil,
			"Wrong action type %T, expected HTTPAction", a)
	}
	return httpAction.Request, nil
}
