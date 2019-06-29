// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package http_action

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mattermost/mattermost-plugin-jira/server/action"
	"github.com/pkg/errors"
)

type Action struct {
	action.Action

	r      *http.Request
	rw     http.ResponseWriter
	status int
}

var _ action.Action = (*Action)(nil)

func Make(router action.Router, inner action.Action, r *http.Request, w http.ResponseWriter) action.Action {
	a := &Action{
		Action: inner,
		r:      r,
		rw:     w,
	}
	a.Context().MattermostUserId = r.Header.Get("Mattermost-User-Id")
	return a
}

func (a Action) FormValue(key string) string {
	return a.r.FormValue(key)
}

func (a Action) RespondError(status int, err error, wrap ...interface{}) error {
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

	a.status = status
	http.Error(a.rw, err.Error(), status)
	return err
}

func (a Action) RespondPrintf(format string, args ...interface{}) error {
	text := fmt.Sprintf(format, args...)
	a.rw.Header().Set("Content-Type", "text/plain")
	_, err := a.rw.Write([]byte(text))
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err,
			"failed to write response")
	}
	return nil
}

func (a Action) RespondRedirect(redirectURL string) error {
	status := http.StatusFound
	if a.r.Method != http.MethodGet {
		status = http.StatusTemporaryRedirect
	}
	http.Redirect(a.rw, a.r, redirectURL, status)
	a.status = status
	return nil
}

func (a Action) RespondTemplate(templateKey, contentType string, values interface{}) error {
	t := a.Context().Templates[templateKey]
	if t == nil {
		return a.RespondError(http.StatusInternalServerError, nil,
			"no template found for %q", templateKey)
	}
	a.rw.Header().Set("Content-Type", contentType)
	err := t.Execute(a.rw, values)
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err,
			"failed to write response")
	}
	return nil
}

func (a Action) RespondJSON(value interface{}) error {
	a.rw.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(a.rw).Encode(value)
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err,
			"failed to write response")
	}
	return nil
}

func Require(a action.Action) error {
	_, ok := a.(*Action)
	if !ok {
		return errors.Errorf("wrong action type, expected HTTPAction, got %T", a)
	}
	return nil
}

func RespondTemplate(action action.Action, contentType string, values interface{}) error {
	a, _ := action.(*Action)
	return a.RespondTemplate(a.r.URL.Path, contentType, values)
}

func Request(action action.Action) (*http.Request, error) {
	a, _ := action.(*Action)
	return a.r, nil
}

func LogAction(a action.Action) error {
	httpAction, ok := a.(*Action)
	switch {
	case !ok:
		a.Errorf("http: error: misconfiguration, wrong action type %T", a)
	case a.Context().LogErr != nil:
		a.Infof("http: %v %v, error:%v", httpAction.status, httpAction.r.URL, a.Context().LogErr)
	default:
		a.Debugf("http: %v %v", httpAction.status, httpAction.r.URL)
	}
	return nil
}
