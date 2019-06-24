// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package http

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/mattermost/mattermost-plugin-jira/server/jira"

	"github.com/mattermost/mattermost-plugin-jira/server/action"
)

func processSubscribeWebhook(a action.Action) error {
	ac := a.Context()
	httpAction, ok := a.(*action.HTTPAction)
	if !ok {
		return a.RespondError(http.StatusInternalServerError, nil, "misconfiguration, wrong Action type")
	}

	if subtle.ConstantTimeCompare(
		[]byte(httpAction.Request.URL.Query().Get("secret")),
		[]byte(ac.WebhookSecret)) != 1 {
		return a.RespondError(http.StatusForbidden, nil,
			"request URL: secret did not match")
	}

	status, err := app.ProcessSubscribeWebhook(ac.API, ac.UserStore, httpAction.Request.Body, ac.BotUserId)
	if err != nil {
		return a.RespondError(status, err)
	}
	return a.RespondJSON(map[string]string{"Status": "OK"})
}

func handleChannelSubscriptions(a action.Action) error {
	request, err := action.HTTPRequest(a)
	if err != nil {
		return err
	}
	ac := a.Context()

	switch request.Method {
	case http.MethodPost:
		return createChannelSubscription(a, ac, request)
	case http.MethodDelete:
		return deleteChannelSubscription(a, ac, request)
	case http.MethodGet:
		return getChannelSubscriptions(a, ac, request)
	case http.MethodPut:
		return editChannelSubscription(a, ac, request)
	default:
		return a.RespondError(http.StatusMethodNotAllowed, nil, "Request: %q is not allowed.", request.Method)
	}
}

func createChannelSubscription(a action.Action, ac *action.Context, request *http.Request) error {
	status, err := app.CreateChannelSubscription(ac.API, ac.MattermostUserId, request.Body)
	if err != nil {
		return a.RespondError(status, err,
			"failed to create a channel subscription")
	}
	return a.RespondJSON(map[string]string{"status": "OK"})
}

func editChannelSubscription(a action.Action, ac *action.Context, request *http.Request) error {
	status, err := app.EditChannelSubscription(ac.API, ac.MattermostUserId, request.Body)
	if err != nil {
		return a.RespondError(status, err,
			"failed to create a channel subscription")
	}

	return a.RespondJSON(map[string]string{"status": "OK"})
}

func deleteChannelSubscription(a action.Action, ac *action.Context, request *http.Request) error {
	// routeAPISubscriptionsChannel has the trailing '/'
	subscriptionId := strings.TrimPrefix(request.URL.Path, routeAPIChannelSubscriptions)
	if len(subscriptionId) != 26 {
		return a.RespondError(http.StatusBadRequest, nil,
			"bad subscription id")
	}

	status, err := app.DeleteChannelSubscription(ac.API, ac.MattermostUserId, subscriptionId)
	if err != nil {
		return a.RespondError(status, err,
			"failed to create a channel subscription")
	}
	return a.RespondJSON(map[string]string{"status": "OK"})
}

func getChannelSubscriptions(a action.Action, ac *action.Context, request *http.Request) error {
	// routeAPISubscriptionsChannel has the trailing '/'
	channelId := strings.TrimPrefix(request.URL.Path, routeAPIChannelSubscriptions)
	if len(channelId) != 26 {
		return a.RespondError(http.StatusBadRequest, nil,
			"bad channel id")
	}

	subscriptions, status, err := app.GetChannelSubscriptions(ac.API, ac.MattermostUserId, channelId)
	if err != nil {
		return a.RespondError(status, err,
			"failed to create a channel subscription")
	}

	return a.RespondJSON(subscriptions)
}
