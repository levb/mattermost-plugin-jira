package proxy

import (
	"net/http"

	"github.com/mattermost/mattermost-server/model"

	"github.com/mattermost/mattermost-plugin-jira/server/action"
)

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
	return nil
}

func RequireUpstreamUser(a action.Action) error {
	ac := a.Context()
	if ac.UpstreamUser != nil {
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
	a.Debugf("action: loaded upstream user %q", user.UpstreamDisplayName())
	ac.UpstreamUser = user
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
	a.Debugf("action: loaded upstream %q", up.Config().Key)
	return nil
}
