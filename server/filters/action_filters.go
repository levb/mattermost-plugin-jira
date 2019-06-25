package filters

import (
	"net/http"

	"github.com/mattermost/mattermost-plugin-jira/server/jira/jiracloud"

	"github.com/mattermost/mattermost-plugin-jira/server/upstream"

	"github.com/dgrijalva/jwt-go"
	"github.com/mattermost/mattermost-server/model"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-jira/server/action"
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
	httpAction, ok := a.(*action.HTTPAction)
	if !ok {
		return a.RespondError(http.StatusInternalServerError, nil,
			"Wrong action type %T, eexpected HTTPAction", a)
	}
	if httpAction.Request.Method != method {
		return a.RespondError(http.StatusMethodNotAllowed, nil,
			"method %s is not allowed, must be %s", httpAction.Request.Method, method)
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
	err := action.Script{
		RequireMattermostUserId,
	}.Run(a)
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
	err := action.Script{
		RequireMattermostUser,
	}.Run(a)
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

func RequireBackendUser(a action.Action) error {
	ac := a.Context()
	if ac.User != nil {
		return nil
	}
	err := action.Script{
		RequireMattermostUserId,
		RequireUpstream,
	}.Run(a)
	if err != nil {
		return err
	}

	user, err := ac.UserStore.Load(ac.MattermostUserId)
	if err != nil {
		return a.RespondError(http.StatusUnauthorized, err)
	}
	a.Debugf("action: loaded Jira user %q", user.DisplayName)
	ac.User = user
	return nil
}

func RequireJiraClient(a action.Action) error {
	ac := a.Context()
	if ac.JiraClient != nil {
		return nil
	}
	err := action.Script{
		RequireUpstream,
		RequireBackendUser,
	}.Run(a)
	if err != nil {
		return err
	}

	jiraClient, err := ac.Upstream.GetClient(ac.PluginId, ac.User)
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err)
	}
	ac.JiraClient = jiraClient
	a.Debugf("action: loaded Jira client for %q", ac.User.DisplayName)
	return nil
}

func RequireUpstream(a action.Action) error {
	ac := a.Context()
	if ac.Upstream != nil {
		return nil
	}
	be, err := ac.UpstreamLoader.Current()
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err)
	}
	ac.Upstream = be

	// Important: overwrite the default UserStore with that where
	// the keys are prefixed with the instance URL
	ac.UserStore = instance.NewUserStore(ac.UserStore, be)
	a.Debugf("action: loaded Jira instance %q", be.GetURL())
	return nil
}

func RequireJiraCloudJWT(a action.Action) error {
	ac := a.Context()
	if ac.BackendJWT != nil {
		return nil
	}
	err := action.Script{
		RequireUpstream,
	}.Run(a)
	if err != nil {
		return err
	}

	tokenString := a.FormValue("jwt")
	if tokenString == "" {
		return a.RespondError(http.StatusBadRequest, nil,
			"no jwt found in the HTTP request")
	}

	cloudUpstream, ok := ac.Upstream.(*jira_cloud.Upstream)
	if !ok {
		return a.RespondError(http.StatusInternalServerError, nil,
			"misconfigured instance type")
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.Errorf(
				"unsupported signing method: %v", token.Header["alg"])
		}
		// HMAC secret is a []byte
		return []byte(cloudUpstream.AtlassianSecurityContext.SharedSecret), nil
	})
	if err != nil || !token.Valid {
		return a.RespondError(http.StatusUnauthorized, err,
			"failed to validate JWT")
	}

	ac.BackendJWT = token
	ac.BackendRawJWT = tokenString
	a.Debugf("action: verified Jira JWT")
	return nil
}
