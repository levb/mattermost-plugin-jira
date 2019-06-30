// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package http

import (
	"crypto/subtle"
	"math"
	"net/http"
	"net/url"

	"github.com/mattermost/mattermost-plugin-jira/server/action"
	"github.com/mattermost/mattermost-plugin-jira/server/action/http_action"
	"github.com/mattermost/mattermost-plugin-jira/server/jira"
)

const maskLegacy = jira.WebhookEventCreated |
	jira.WebhookEventUpdatedReopened |
	jira.WebhookEventUpdatedResolved |
	jira.WebhookEventDeletedUnresolved

const maskComments = jira.WebhookEventCreatedComment |
	jira.WebhookEventDeletedComment |
	jira.WebhookEventUpdatedComment

const maskDefault = maskLegacy |
	jira.WebhookEventUpdatedAssignee |
	maskComments

const maskAll = math.MaxUint64

// The keys listed here can be used in the Jira webhook URL to control what events
// are posted to Mattermost. A matching parameter with a non-empty value must
// be added to turn on the event display.
var eventParamMasks = map[string]uint64{
	"updated_attachment":  jira.WebhookEventUpdatedAttachment,  // updated attachments
	"updated_description": jira.WebhookEventUpdatedDescription, // issue description edited
	"updated_labels":      jira.WebhookEventUpdatedLabels,      // updated labels
	"updated_prioity":     jira.WebhookEventUpdatedPriority,    // changes in priority
	"updated_rank":        jira.WebhookEventUpdatedRank,        // ranked higher or lower
	"updated_sprint":      jira.WebhookEventUpdatedSprint,      // assigned to a different sprint
	"updated_status":      jira.WebhookEventUpdatedStatus,      // transitions like Done, In Progress
	"updated_summary":     jira.WebhookEventUpdatedSummary,     // issue renamed
	"updated_all":         maskAll,                             // all events
}

func processLegacyWebhook(a action.Action) error {
	ac := a.Context()
	r := http_action.Request(a)

	if ac.WebhookSecret == "" {
		return a.RespondError(http.StatusInternalServerError, nil,
			"webhook secret not configured")
	}

	secret := a.FormValue("secret")
	// secret may be URL-escaped, potentially more than once. Loop until there
	// are no % escapes left.
	for {
		if subtle.ConstantTimeCompare([]byte(secret), []byte(ac.WebhookSecret)) == 1 {
			break
		}

		unescaped, _ := url.QueryUnescape(secret)
		if unescaped == secret {
			return a.RespondError(http.StatusForbidden, nil, "Request URL: secret did not match")
		}
		secret = unescaped
	}
	teamName := a.FormValue("team")
	if teamName == "" {
		return a.RespondError(http.StatusBadRequest, nil, "Request URL: no team name found")
	}
	channelName := a.FormValue("channel")
	if channelName == "" {
		return a.RespondError(http.StatusBadRequest, nil, "Request URL: no channel name found")
	}
	eventMask := maskDefault
	for key, paramMask := range eventParamMasks {
		if a.FormValue(key) == "" {
			continue
		}
		eventMask = eventMask | paramMask
	}

	channel, appErr := ac.API.GetChannelByNameForTeamName(teamName, channelName, false)
	if appErr != nil {
		return a.RespondError(appErr.StatusCode, appErr)
	}

	wh, _, err := jira.ParseWebhook(r.Body)
	if err != nil {
		return a.RespondError(http.StatusBadRequest, err)
	}

	wh.PostNotifications(ac.API, ac.Upstream, ac.BotUserId)
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err)
	}

	// Skip events we don't need to post
	if eventMask&wh.EventMask() == 0 {
		return nil
	}

	// Post the event to the subscribed channel
	_, statusCode, err := wh.PostToChannel(ac.API, channel.Id, ac.BotUserId)
	if err != nil {
		return a.RespondError(statusCode, err)
	}

	return nil
}
