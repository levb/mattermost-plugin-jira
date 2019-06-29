package lib

import (
	"net/http"

	"github.com/mattermost/mattermost-server/model"

	"github.com/mattermost/mattermost-plugin-jira/server/action"
	"github.com/mattermost/mattermost-plugin-jira/server/action/http_action"
)

func RequireHTTPGet(a action.Action) error {
	return requireHTTPMethod(a, http.MethodGet)
}

func RequireHTTPPost(a action.Action) error {
	return requireHTTPMethod(a, http.MethodPost)
}

func RequireHTTPPut(a action.Action) error {
	return requireHTTPMethod(a, http.MethodPut)
}

func RequireHTTPDelete(a action.Action) error {
	return requireHTTPMethod(a, http.MethodDelete)
}

func requireHTTPMethod(a action.Action, method string) error {
	r, err := http_action.Request(a)
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err)
	}
	if r.Method != method {
		return a.RespondError(http.StatusMethodNotAllowed, nil,
			"method %s is not allowed, must be %s", r.Method, method)
	}
	return nil
}

func RequireMattermostUserId(a action.Action) error {
	ac := a.Context()
	if ac.MattermostUserId == "" {
		return a.RespondError(http.StatusUnauthorized, nil,
			"not authorized")
	}
	// MattermostUserId is set by the protocol-specific New...Action, nothing to do here
	return nil
}

func RequireMattermostUser(a action.Action) error {
	ac := a.Context()
	if ac.MattermostUser != nil {
		return nil
	}
	err := action.Script{RequireMattermostUserId}.Run(a)
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

func RequireMattermostSysAdmin(a action.Action) error {
	err := action.Script{RequireMattermostUser}.Run(a)
	if err != nil {
		return err
	}

	ac := a.Context()
	if !ac.MattermostUser.IsInRole(model.SYSTEM_ADMIN_ROLE_ID) {
		return a.RespondError(http.StatusUnauthorized, nil,
			"reserverd for system administrators")
	}
	// if !ac.API.HasPermissionTo(ac.MattermostUserId, model.PERMISSION_MANAGE_SYSTEM) {
	// 	return a.RespondError(http.StatusForbidden, nil, "forbidden")
	// }

	return nil
}

func RequireUpstreamUser(a action.Action) error {
	ac := a.Context()
	if ac.User != nil {
		return nil
	}
	err := action.Script{RequireMattermostUserId, RequireUpstream}.Run(a)
	if err != nil {
		return err
	}

	user, err := ac.Upstream.LoadUser(ac.MattermostUserId)
	if err != nil {
		return a.RespondError(http.StatusUnauthorized, err)
	}
	a.Debugf("action: loaded Jira user %q", user.UpstreamDisplayName())
	ac.User = user
	return nil
}

func RequireUpstream(a action.Action) error {
	ac := a.Context()
	if ac.Upstream != nil {
		return nil
	}
	up, err := ac.UpstreamStore.LoadCurrent()
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err)
	}
	ac.Upstream = up

	// Important: overwrite the default UserStore with that where
	// the keys are prefixed with the instance URL
	a.Debugf("action: loaded Jira instance %q", up.Config().Key)
	return nil
}

func RequireUpstreamClient(a action.Action) error {
	ac := a.Context()
	if ac.JiraClient != nil {
		return nil
	}
	err := action.Script{RequireUpstream, RequireUpstreamUser}.Run(a)
	if err != nil {
		return err
	}

	client, err := ac.Upstream.GetClient(ac.PluginId, ac.User)
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err)
	}
	ac.JiraClient = client
	a.Debugf("action: loaded Jira client for %q", ac.User.UpstreamDisplayName())
	return nil
}
