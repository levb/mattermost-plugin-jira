// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package http

import (
	"net/http"

	"github.com/mattermost/mattermost-plugin-jira/server/action"
	"github.com/mattermost/mattermost-plugin-jira/server/jira"
)

func connectUser(a action.Action) error {
	ac := a.Context()

	redirectURL, status, err := app.GetUserConnectURL(
		ac.UserStore, ac.OneTimeStore, ac.Upstream, ac.PluginURL, ac.MattermostUserId)
	if err != nil {
		return a.RespondError(status, err)
	}
	return a.RespondRedirect(redirectURL)
}

func disconnectUser(a action.Action) error {
	ac := a.Context()
	err := app.DeleteUserNotify(ac.API, ac.UserStore, ac.MattermostUserId)
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err)
	}

	// TODO replace with template
	return a.RespondPrintf(`
<!DOCTYPE html>
<html>
       <head>
               <script>
                       // window.close();
               </script>
       </head>
       <body>
               <p>Disconnected from Jira. Please close this page.</p>
       </body>
</html>
`)
}

func getUserInfo(a action.Action) error {
	ac := a.Context()
	return a.RespondJSON(app.GetUserInfo(ac.UpstreamLoader,
		ac.UserStore,
		ac.MattermostUserId,
	))
}
