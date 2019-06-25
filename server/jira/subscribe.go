// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package jira

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/mattermost/mattermost-plugin-jira/server/upstream"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
	"github.com/pkg/errors"
)

const (
	JIRA_WEBHOOK_EVENT_ISSUE_CREATED = "jira:issue_created"
	JIRA_WEBHOOK_EVENT_ISSUE_UPDATED = "jira:issue_updated"
	JIRA_WEBHOOK_EVENT_ISSUE_DELETED = "jira:issue_deleted"
)

const JiraSubscriptionsKey = "jirasub"

type ChannelSubscription struct {
	Id        string              `json:"id"`
	ChannelId string              `json:"channel_id"`
	Filters   map[string][]string `json:"filters"`
}

type ChannelSubscriptions struct {
	ById          map[string]ChannelSubscription `json:"by_id"`
	IdByChannelId map[string][]string            `json:"id_by_channel_id"`
	IdByEvent     map[string][]string            `json:"id_by_event"`
}

func NewChannelSubscriptions() *ChannelSubscriptions {
	return &ChannelSubscriptions{
		ById:          map[string]ChannelSubscription{},
		IdByChannelId: map[string][]string{},
		IdByEvent:     map[string][]string{},
	}
}

func (s *ChannelSubscriptions) remove(sub *ChannelSubscription) {
	delete(s.ById, sub.Id)

	remove := func(ids []string, idToRemove string) []string {
		for i, id := range ids {
			if id == idToRemove {
				ids[i] = ids[len(ids)-1]
				return ids[:len(ids)-1]
			}
		}
		return ids
	}

	s.IdByChannelId[sub.ChannelId] = remove(s.IdByChannelId[sub.ChannelId], sub.Id)

	for _, event := range sub.Filters["events"] {
		s.IdByEvent[event] = remove(s.IdByEvent[event], sub.Id)
	}
}

func (s *ChannelSubscriptions) add(newSubscription *ChannelSubscription) {
	s.ById[newSubscription.Id] = *newSubscription
	s.IdByChannelId[newSubscription.ChannelId] = append(s.IdByChannelId[newSubscription.ChannelId], newSubscription.Id)
	for _, event := range newSubscription.Filters["events"] {
		s.IdByEvent[event] = append(s.IdByEvent[event], newSubscription.Id)
	}
}

type Subscriptions struct {
	Channel *ChannelSubscriptions
}

func NewSubscriptions() *Subscriptions {
	return &Subscriptions{
		Channel: NewChannelSubscriptions(),
	}
}

func SubscriptionsFromJson(bytes []byte) (*Subscriptions, error) {
	var subs *Subscriptions
	if len(bytes) != 0 {
		unmarshalErr := json.Unmarshal(bytes, &subs)
		if unmarshalErr != nil {
			return nil, unmarshalErr
		}
	} else {
		subs = NewSubscriptions()
	}

	return subs, nil
}

func getChannelsSubscribed(api plugin.API, jwh *JiraWebhook) ([]string, error) {
	subs, err := getSubscriptions(api)
	if err != nil {
		return nil, err
	}

	subIds := subs.Channel.IdByEvent[jwh.WebhookEvent]

	channelIds := []string{}
	for _, subId := range subIds {
		sub := subs.Channel.ById[subId]

		acceptable := true
		for field, acceptableValues := range sub.Filters {
			// Blank in acceptable values means all values are acceptable
			if len(acceptableValues) == 0 {
				continue
			}
			switch field {
			case "event":
				found := false
				for _, acceptableEvent := range acceptableValues {
					if acceptableEvent == jwh.WebhookEvent {
						found = true
						break
					}
				}
				if !found {
					acceptable = false
					break
				}
			case "project":
				found := false
				for _, acceptableProject := range acceptableValues {
					if acceptableProject == jwh.Issue.Fields.Project.Key {
						found = true
						break
					}
				}
				if !found {
					acceptable = false
					break
				}
			case "issue_type":
				found := false
				for _, acceptableIssueType := range acceptableValues {
					if acceptableIssueType == jwh.Issue.Fields.Type.ID {
						found = true
						break
					}
				}
				if !found {
					acceptable = false
					break
				}
			}
		}

		if acceptable {
			channelIds = append(channelIds, sub.ChannelId)
		}
	}

	return channelIds, nil
}

func getSubscriptions(api plugin.API) (*Subscriptions, error) {
	data, err := api.KVGet(JiraSubscriptionsKey)
	if err != nil {
		return nil, err
	}
	return SubscriptionsFromJson(data)
}

func getSubscriptionsForChannel(api plugin.API, channelId string) ([]ChannelSubscription, error) {
	subs, err := getSubscriptions(api)
	if err != nil {
		return nil, err
	}

	channelSubscriptions := []ChannelSubscription{}
	for _, channelSubscriptionId := range subs.Channel.IdByChannelId[channelId] {
		channelSubscriptions = append(channelSubscriptions, subs.Channel.ById[channelSubscriptionId])
	}

	return channelSubscriptions, nil
}

func getChannelSubscription(api plugin.API, subscriptionId string) (*ChannelSubscription, error) {
	subs, err := getSubscriptions(api)
	if err != nil {
		return nil, err
	}

	subscription, ok := subs.Channel.ById[subscriptionId]
	if !ok {
		return nil, errors.New("could not find subscription")
	}

	return &subscription, nil
}

func removeChannelSubscription(api plugin.API, subscriptionId string) error {
	return atomicModify(api, JiraSubscriptionsKey, func(initialBytes []byte) ([]byte, error) {
		subs, err := SubscriptionsFromJson(initialBytes)
		if err != nil {
			return nil, err
		}

		subscription, ok := subs.Channel.ById[subscriptionId]
		if !ok {
			return nil, errors.New("could not find subscription")
		}

		subs.Channel.remove(&subscription)

		modifiedBytes, marshalErr := json.Marshal(&subs)
		if marshalErr != nil {
			return nil, marshalErr
		}

		return modifiedBytes, nil
	})
}

func addChannelSubscription(api plugin.API, newSubscription *ChannelSubscription) error {
	return atomicModify(api, JiraSubscriptionsKey, func(initialBytes []byte) ([]byte, error) {
		subs, err := SubscriptionsFromJson(initialBytes)
		if err != nil {
			return nil, err
		}

		newSubscription.Id = model.NewId()
		subs.Channel.add(newSubscription)

		modifiedBytes, marshalErr := json.Marshal(&subs)
		if marshalErr != nil {
			return nil, marshalErr
		}

		return modifiedBytes, nil
	})
}

func editChannelSubscription(api plugin.API, modifiedSubscription *ChannelSubscription) error {
	return atomicModify(api, JiraSubscriptionsKey, func(initialBytes []byte) ([]byte, error) {
		subs, err := SubscriptionsFromJson(initialBytes)
		if err != nil {
			return nil, err
		}

		oldSub, ok := subs.Channel.ById[modifiedSubscription.Id]
		if !ok {
			return nil, errors.New("Existing subscription does not exist.")
		}
		subs.Channel.remove(&oldSub)
		subs.Channel.add(modifiedSubscription)

		modifiedBytes, marshalErr := json.Marshal(&subs)
		if marshalErr != nil {
			return nil, marshalErr
		}

		return modifiedBytes, nil
	})
}

func atomicModify(api plugin.API, key string, modify func(initialValue []byte) ([]byte, error)) error {
	readModify := func() ([]byte, []byte, error) {
		initialBytes, appErr := api.KVGet(key)
		if appErr != nil {
			return nil, nil, errors.Wrap(appErr, "unable to read inital value")
		}

		modifiedBytes, err := modify(initialBytes)
		if err != nil {
			return nil, nil, errors.Wrap(err, "modification error")
		}

		return initialBytes, modifiedBytes, nil
	}

	success := false
	for !success {
		//initialBytes, newValue, err := readModify()
		_, newValue, err := readModify()
		if err != nil {
			return err
		}

		var setError *model.AppError
		// Commenting this out so we can support < 5.12 for 2.0
		//success, setError = p.API.KVCompareAndSet(key, initialBytes, newValue)
		setError = api.KVSet(key, newValue)
		success = true
		if setError != nil {
			return errors.Wrap(setError, "problem writing value")
		}

	}

	return nil
}

func ProcessSubscribeWebhook(api plugin.API, userStore upstream.UserStore, body io.Reader, botUserId string) (int, error) {
	var err error
	var status int
	wh, jwh, err := ParseWebhook(body)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	channelIds, err := getChannelsSubscribed(api, jwh)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	for _, channelId := range channelIds {
		_, status, err = wh.PostToChannel(api, channelId, botUserId)
		if err != nil {
			return status, err
		}
	}

	_, status, err = wh.PostNotifications(api, userStore, botUserId)
	if err != nil {
		return status, err
	}

	return http.StatusOK, nil
}

func CreateChannelSubscription(api plugin.API, mattermostUserId string, body io.Reader) (int, error) {
	subscription := ChannelSubscription{}
	err := json.NewDecoder(body).Decode(&subscription)
	if err != nil {
		return http.StatusBadRequest,
			errors.WithMessage(err, "failed to decode incoming request")
	}

	if len(subscription.ChannelId) != 26 ||
		len(subscription.Id) != 0 {
		return http.StatusBadRequest, errors.New("Channel subscription invalid")
	}

	_, appErr := api.GetChannelMember(subscription.ChannelId, mattermostUserId)
	if appErr != nil {
		return http.StatusForbidden, errors.New("Not a member of the channel specified")
	}

	if err := addChannelSubscription(api, &subscription); err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func EditChannelSubscription(api plugin.API, mattermostUserId string, body io.Reader) (int, error) {
	subscription := ChannelSubscription{}
	err := json.NewDecoder(body).Decode(&subscription)
	if err != nil {
		return http.StatusBadRequest,
			errors.WithMessage(err, "failed to decode incoming request")
	}

	if len(subscription.ChannelId) != 26 ||
		len(subscription.Id) != 26 {
		return http.StatusBadRequest, errors.New("Channel subscription invalid")
	}

	if _, appErr := api.GetChannelMember(subscription.ChannelId, mattermostUserId); appErr != nil {
		return http.StatusForbidden,
			errors.New("Not a member of the channel specified")
	}

	if err := editChannelSubscription(api, &subscription); err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func DeleteChannelSubscription(api plugin.API, mattermostUserId, subscriptionId string) (int, error) {
	subscription, err := getChannelSubscription(api, subscriptionId)
	if err != nil {
		return http.StatusBadRequest,
			errors.WithMessage(err, "bad subscription id")
	}

	_, appErr := api.GetChannelMember(subscription.ChannelId, mattermostUserId)
	if appErr != nil {
		return http.StatusForbidden,
			errors.New("Not a member of the channel specified")
	}

	if err := removeChannelSubscription(api, subscriptionId); err != nil {
		return http.StatusInternalServerError,
			errors.WithMessage(err, "unable to remove channel subscription")
	}

	return http.StatusOK, nil
}

func GetChannelSubscriptions(api plugin.API, mattermostUserId, channelId string) ([]ChannelSubscription, int, error) {
	if _, appErr := api.GetChannelMember(channelId, mattermostUserId); appErr != nil {
		return nil, http.StatusForbidden, errors.New("Not a member of the channel specified")
	}

	subscriptions, err := getSubscriptionsForChannel(api, channelId)
	if err != nil {
		return nil, http.StatusInternalServerError,
			errors.WithMessage(err, "unable to get channel subscriptions")
	}

	return subscriptions, http.StatusOK, nil
}
