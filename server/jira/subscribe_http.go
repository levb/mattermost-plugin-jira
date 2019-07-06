// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package jira

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/mattermost/mattermost-plugin-jira/server/action"
	"github.com/mattermost/mattermost-plugin-jira/server/action/http_action"
	"github.com/mattermost/mattermost-plugin-jira/server/proxy"
)

const (
	routeAPIChannelSubscriptions = "/api/v2/subscriptions/channel/" // trailing '/' on purpose
	routeAPISubscribeWebhook     = "/api/v2/webhook"
)

var subscribeHTTPRoutes = map[string]*action.Route{
	routeAPISubscribeWebhook: action.NewRoute(
		proxy.RequireHTTPMethod(http.MethodPost),
		proxy.RequireUpstream,
		httpSubscribeWebhook,
	),
	// httpChannelSubscriptions already ends in a '/', so adding "*" will
	// pass all sub-paths up to the handler
	routeAPIChannelSubscriptions + "*": action.NewRoute(
		proxy.RequireMattermostUserId,
		httpChannelSubscriptions,
	),
}

func httpSubscribeWebhook(a action.Action) error {
	ac := a.Context()
	r := http_action.Request(a)

	if subtle.ConstantTimeCompare(
		[]byte(r.URL.Query().Get("secret")),
		[]byte(ac.PluginWebhookSecret)) != 1 {
		return a.RespondError(http.StatusForbidden, nil,
			"request URL: secret did not match")
	}

	status, err := processSubscribeWebhook(ac.API, ac.Upstream, r.Body, ac.BotUserId)
	if err != nil {
		return a.RespondError(status, err)
	}
	return a.RespondJSON(map[string]string{"Status": "OK"})
}

func httpChannelSubscriptions(a action.Action) error {
	r := http_action.Request(a)

	switch r.Method {
	case http.MethodPost:
		return httpCreateChannelSubscription(a)
	case http.MethodDelete:
		return httpDeleteChannelSubscription(a)
	case http.MethodGet:
		return httpGetChannelSubscriptions(a)
	case http.MethodPut:
		return httpEditChannelSubscription(a)
	default:
		return a.RespondError(http.StatusMethodNotAllowed, nil, "Request: %q is not allowed.", r.Method)
	}
}

func httpCreateChannelSubscription(a action.Action) error {
	ac := a.Context()
	r := http_action.Request(a)
	status, err := createChannelSubscription(ac.API, ac.MattermostUserId, r.Body)
	if err != nil {
		return a.RespondError(status, err,
			"failed to create a channel subscription")
	}
	return a.RespondJSON(map[string]string{"status": "OK"})
}

func httpEditChannelSubscription(a action.Action) error {
	ac := a.Context()
	r := http_action.Request(a)
	status, err := editChannelSubscription(ac.API, ac.MattermostUserId, r.Body)
	if err != nil {
		return a.RespondError(status, err,
			"failed to create a channel subscription")
	}

	return a.RespondJSON(map[string]string{"status": "OK"})
}

func httpDeleteChannelSubscription(a action.Action) error {
	ac := a.Context()
	r := http_action.Request(a)
	// routeAPISubscriptionsChannel has the trailing '/'
	subscriptionId := strings.TrimPrefix(r.URL.Path, routeAPIChannelSubscriptions)
	if len(subscriptionId) != 26 {
		return a.RespondError(http.StatusBadRequest, nil,
			"bad subscription id")
	}

	status, err := deleteChannelSubscription(ac.API, ac.MattermostUserId, subscriptionId)
	if err != nil {
		return a.RespondError(status, err,
			"failed to create a channel subscription")
	}
	return a.RespondJSON(map[string]string{"status": "OK"})
}

func httpGetChannelSubscriptions(a action.Action) error {
	ac := a.Context()
	r := http_action.Request(a)
	// routeAPISubscriptionsChannel has the trailing '/'
	channelId := strings.TrimPrefix(r.URL.Path, routeAPIChannelSubscriptions)
	if len(channelId) != 26 {
		return a.RespondError(http.StatusBadRequest, nil,
			"bad channel id")
	}

	subscriptions, status, err := getChannelSubscriptions(ac.API, ac.MattermostUserId, channelId)
	if err != nil {
		return a.RespondError(status, err,
			"failed to create a channel subscription")
	}

	return a.RespondJSON(subscriptions)
}
