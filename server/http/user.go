// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package http

import (
	"net/http"

	"github.com/mattermost/mattermost-plugin-jira/server/action"
	"github.com/mattermost/mattermost-plugin-jira/server/jira"
	"github.com/mattermost/mattermost-plugin-jira/server/lib"
)

func connectUser(a action.Action) error {
	ac := a.Context()

	redirectURL, status, err := lib.GetUserConnectURL(ac.PluginURL,
		ac.OneTimeStore, ac.Upstream, ac.MattermostUserId)
	if err != nil {
		return a.RespondError(status, err)
	}
	return a.RespondRedirect(redirectURL)
}

func disconnectUser(a action.Action) error {
	ac := a.Context()
	err := lib.DeleteUserNotify(ac.API, ac.Upstream, ac.User)
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
	resp := jira.GetUserInfo(ac.UpstreamStore, ac.MattermostUserId)

	return a.RespondJSON(resp)
}
