// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package proxy

import (
	"net/http"

	"github.com/mattermost/mattermost-plugin-jira/server/action"
	"github.com/mattermost/mattermost-plugin-jira/server/action/http_action"
)

func RequireHTTPMethod(method string) action.Func {
	return func(a action.Action) error {
		r := http_action.Request(a)
		if r.Method != method {
			return a.RespondError(http.StatusMethodNotAllowed, nil,
				"method %s is not allowed, must be %s", r.Method, method)
		}
		return nil
	}
}
