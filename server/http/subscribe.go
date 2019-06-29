// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package http

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/mattermost/mattermost-plugin-jira/server/action"
	"github.com/mattermost/mattermost-plugin-jira/server/action/http_action"
	"github.com/mattermost/mattermost-plugin-jira/server/jira"
)

func processSubscribeWebhook(a action.Action) error {
	ac := a.Context()
	r, err := http_action.Request(a)
	if err != nil {
		return err
	}

	if subtle.ConstantTimeCompare(
		[]byte(r.URL.Query().Get("secret")),
		[]byte(ac.WebhookSecret)) != 1 {
		return a.RespondError(http.StatusForbidden, nil,
			"request URL: secret did not match")
	}

	status, err := jira.ProcessSubscribeWebhook(ac.API, ac.Upstream, r.Body, ac.BotUserId)
	if err != nil {
		return a.RespondError(status, err)
	}
	return a.RespondJSON(map[string]string{"Status": "OK"})
}

func handleChannelSubscriptions(a action.Action) error {
	ac := a.Context()
	r, err := http_action.Request(a)
	if err != nil {
		return err
	}

	switch r.Method {
	case http.MethodPost:
		return createChannelSubscription(a, ac, r)
	case http.MethodDelete:
		return deleteChannelSubscription(a, ac, r)
	case http.MethodGet:
		return getChannelSubscriptions(a, ac, r)
	case http.MethodPut:
		return editChannelSubscription(a, ac, r)
	default:
		return a.RespondError(http.StatusMethodNotAllowed, nil, "Request: %q is not allowed.", r.Method)
	}
}

func createChannelSubscription(a action.Action, ac *action.Context, request *http.Request) error {
	status, err := jira.CreateChannelSubscription(ac.API, ac.MattermostUserId, request.Body)
	if err != nil {
		return a.RespondError(status, err,
			"failed to create a channel subscription")
	}
	return a.RespondJSON(map[string]string{"status": "OK"})
}

func editChannelSubscription(a action.Action, ac *action.Context, request *http.Request) error {
	status, err := jira.EditChannelSubscription(ac.API, ac.MattermostUserId, request.Body)
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

	status, err := jira.DeleteChannelSubscription(ac.API, ac.MattermostUserId, subscriptionId)
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

	subscriptions, status, err := jira.GetChannelSubscriptions(ac.API, ac.MattermostUserId, channelId)
	if err != nil {
		return a.RespondError(status, err,
			"failed to create a channel subscription")
	}

	return a.RespondJSON(subscriptions)
}
