// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package plugin_tests

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/mattermost/mattermost-plugin-jira/server/jira"
	"github.com/mattermost/mattermost-plugin-jira/server/plugin"
	"github.com/mattermost/mattermost-server/model"
	mmplugin "github.com/mattermost/mattermost-server/plugin"
	"github.com/mattermost/mattermost-server/plugin/plugintest"
	"github.com/mattermost/mattermost-server/plugin/plugintest/mock"
	"github.com/stretchr/testify/assert"
)

func checkNotSubscriptions(subsToCheck []jira.ChannelSubscription, existing *jira.Subscriptions, t *testing.T) func(api *plugintest.API) {
	return func(api *plugintest.API) {
		var existingBytes []byte
		if existing != nil {
			var err error
			existingBytes, err = json.Marshal(existing)
			assert.Nil(t, err)
		}

		api.On("KVGet", jira.SubscriptionsKey).Return(existingBytes, nil)

		// Temp changes to revert when we can use KVCompareAndSet
		//api.On("KVCompareAndSet", jira.SubscriptionsKey, existingBytes, mock.MatchedBy(func(data []byte) bool {
		api.On("KVSet", jira.SubscriptionsKey, mock.MatchedBy(func(data []byte) bool {
			//t.Log(string(data))
			var savedSubs jira.Subscriptions
			err := json.Unmarshal(data, &savedSubs)
			assert.Nil(t, err)

			for _, subToCheck := range subsToCheck {
				for _, savedSub := range savedSubs.Channel.ById {
					if subToCheck.Id == savedSub.Id {
						return false
					}
				}
			}

			return true
			//})).Return(true, nil)
		})).Return(nil)
	}
}

func checkHasSubscriptions(subsToCheck []jira.ChannelSubscription, existing *jira.Subscriptions, t *testing.T) func(api *plugintest.API) {
	return func(api *plugintest.API) {
		var existingBytes []byte
		if existing != nil {
			var err error
			existingBytes, err = json.Marshal(existing)
			assert.Nil(t, err)
		}

		api.On("KVGet", jira.SubscriptionsKey).Return(existingBytes, nil)

		// Temp changes to revert when we can use KVCompareAndSet
		//api.On("KVCompareAndSet", JIRA_SUBSCRIPTIONS_KEY, existingBytes, mock.MatchedBy(func(data []byte) bool {
		api.On("KVSet", jira.SubscriptionsKey, mock.MatchedBy(func(data []byte) bool {
			//t.Log(string(data))
			var savedSubs jira.Subscriptions
			err := json.Unmarshal(data, &savedSubs)
			assert.Nil(t, err)

			for _, subToCheck := range subsToCheck {
				var foundSub *jira.ChannelSubscription
				for _, savedSub := range savedSubs.Channel.ById {
					if subToCheck.ChannelId == savedSub.ChannelId && reflect.DeepEqual(subToCheck.Filters, savedSub.Filters) {
						foundSub = &savedSub
						break
					}
				}

				// Check subscription exists
				if foundSub == nil {
					return false
				}

				// Check it's properly attached
				assert.Contains(t, savedSubs.Channel.IdByChannelId[foundSub.ChannelId], foundSub.Id)
				for _, event := range foundSub.Filters["events"] {
					assert.Contains(t, savedSubs.Channel.IdByEvent[event], foundSub.Id)
				}
			}

			return true
			//})).Return(true, nil)
		})).Return(nil)
	}
}

func withExistingChannelSubscriptions(subscriptions []jira.ChannelSubscription) *jira.Subscriptions {
	ret := jira.NewSubscriptions()
	for _, sub := range subscriptions {
		ret.Channel.Add(&sub)
	}
	return ret
}

func hasSubscriptions(subscriptions []jira.ChannelSubscription, t *testing.T) func(api *plugintest.API) {
	return func(api *plugintest.API) {
		subs := withExistingChannelSubscriptions(subscriptions)

		existingBytes, err := json.Marshal(&subs)
		assert.Nil(t, err)

		api.On("KVGet", jira.SubscriptionsKey).Return(existingBytes, nil)
	}
}

func TestSubscribe(t *testing.T) {
	for name, tc := range map[string]struct {
		subscription       string
		expectedStatusCode int
		skipAuthorize      bool
		apiCalls           func(*plugintest.API)
	}{
		"Invalid": {
			subscription:       "{}",
			expectedStatusCode: http.StatusBadRequest,
		},
		"Not Authorized": {
			subscription:       "{}",
			expectedStatusCode: http.StatusUnauthorized,
			skipAuthorize:      true,
		},
		"Won't Decode": {
			subscription:       "{woopsie",
			expectedStatusCode: http.StatusBadRequest,
		},
		"No channel id": {
			subscription:       `{"channel_id": "badchannelid", "fields": {}}`,
			expectedStatusCode: http.StatusBadRequest,
		},
		"Reject Ids": {
			subscription:       `{"id": "iamtryingtodosendid", "channel_id": "aaaaaaaaaaaaaaaaaaaaaaaaaa", "filters": {}}`,
			expectedStatusCode: http.StatusBadRequest,
		},
		"Initial Subscription": {
			subscription:       `{"channel_id": "aaaaaaaaaaaaaaaaaaaaaaaaab", "filters": {"events": ["jira:issue_created"], "project": ["myproject"]}}`,
			expectedStatusCode: http.StatusOK,
			apiCalls: checkHasSubscriptions([]jira.ChannelSubscription{
				jira.ChannelSubscription{
					ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaab",
					Filters: map[string][]string{
						"events":  []string{"jira:issue_created"},
						"project": []string{"myproject"},
					},
				},
			}, nil, t),
		},
		"Adding to existing with other channel": {
			subscription:       `{"channel_id": "aaaaaaaaaaaaaaaaaaaaaaaaab", "filters": {"events": ["jira:issue_created"], "project": ["myproject"]}}`,
			expectedStatusCode: http.StatusOK,
			apiCalls: checkHasSubscriptions([]jira.ChannelSubscription{
				jira.ChannelSubscription{
					ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaab",
					Filters: map[string][]string{
						"events":  []string{"jira:issue_created"},
						"project": []string{"myproject"},
					},
				},
				jira.ChannelSubscription{
					ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaac",
					Filters: map[string][]string{
						"events":  []string{"jira:issue_created"},
						"project": []string{"myproject"},
					},
				},
			},
				withExistingChannelSubscriptions(
					[]jira.ChannelSubscription{
						jira.ChannelSubscription{
							Id:        model.NewId(),
							ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaac",
							Filters: map[string][]string{
								"events":  []string{"jira:issue_created"},
								"project": []string{"myproject"},
							},
						},
					}), t),
		},
		"Adding to existing in same channel": {
			subscription:       `{"channel_id": "aaaaaaaaaaaaaaaaaaaaaaaaab", "filters": {"events": ["jira:issue_created"], "project": ["myproject"]}}`,
			expectedStatusCode: http.StatusOK,
			apiCalls: checkHasSubscriptions([]jira.ChannelSubscription{
				jira.ChannelSubscription{
					ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaab",
					Filters: map[string][]string{
						"events":  []string{"jira:issue_created"},
						"project": []string{"myproject"},
					},
				},
				jira.ChannelSubscription{
					ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaab",
					Filters: map[string][]string{
						"events":  []string{"jira:issue_updated"},
						"project": []string{"myproject"},
					},
				},
			},
				withExistingChannelSubscriptions(
					[]jira.ChannelSubscription{
						jira.ChannelSubscription{
							Id:        model.NewId(),
							ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaab",
							Filters: map[string][]string{
								"events":  []string{"jira:issue_updated"},
								"project": []string{"myproject"},
							},
						},
					}), t),
		},
	} {
		t.Run(name, func(t *testing.T) {
			api := &plugintest.API{}
			p := SetupTestPlugin(t, api, plugin.Config{
				MainConfig: plugin.MainConfig{
					EnableJiraUI:  false,
					WebhookSecret: "somesecret",
					BotUserName:   "someuser",
				},
			}, nil)

			api.On("GetChannelMember", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(&model.ChannelMember{}, (*model.AppError)(nil))

			if tc.apiCalls != nil {
				tc.apiCalls(api)
			}

			w := httptest.NewRecorder()
			request := httptest.NewRequest("POST", "/api/v2/subscriptions/channel", ioutil.NopCloser(bytes.NewBufferString(tc.subscription)))
			if !tc.skipAuthorize {
				request.Header.Set("Mattermost-User-Id", model.NewId())
			}
			p.ServeHTTP(&mmplugin.Context{}, w, request)
			assert.Equal(t, tc.expectedStatusCode, w.Result().StatusCode)
		})
	}
}

func TestDeleteSubscription(t *testing.T) {
	for name, tc := range map[string]struct {
		subscriptionId     string
		expectedStatusCode int
		skipAuthorize      bool
		apiCalls           func(*plugintest.API)
	}{
		"Invalid": {
			subscriptionId:     "blab",
			expectedStatusCode: http.StatusBadRequest,
		},
		"Not Authorized": {
			subscriptionId:     model.NewId(),
			expectedStatusCode: http.StatusUnauthorized,
			skipAuthorize:      true,
		},
		"Sucessfull delete": {
			subscriptionId:     "aaaaaaaaaaaaaaaaaaaaaaaaab",
			expectedStatusCode: http.StatusOK,
			apiCalls: checkNotSubscriptions([]jira.ChannelSubscription{
				jira.ChannelSubscription{
					Id:        "aaaaaaaaaaaaaaaaaaaaaaaaab",
					ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaab",
					Filters: map[string][]string{
						"events":  []string{"jira:issue_created"},
						"project": []string{"myproject"},
					},
				},
			},
				withExistingChannelSubscriptions(
					[]jira.ChannelSubscription{
						jira.ChannelSubscription{
							Id:        "aaaaaaaaaaaaaaaaaaaaaaaaab",
							ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaab",
							Filters: map[string][]string{
								"events":  []string{"jira:issue_created"},
								"project": []string{"myproject"},
							},
						},
						jira.ChannelSubscription{
							Id:        "aaaaaaaaaaaaaaaaaaaaaaaaac",
							ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaab",
							Filters: map[string][]string{
								"events":  []string{"jira:issue_created"},
								"project": []string{"myproject"},
							},
						},
					}), t),
		},
	} {
		t.Run(name, func(t *testing.T) {
			api := &plugintest.API{}
			p := SetupTestPlugin(t, api, plugin.Config{
				MainConfig: plugin.MainConfig{
					EnableJiraUI:  false,
					WebhookSecret: "somesecret",
					BotUserName:   "someuser",
				},
			}, nil)

			api.On("GetChannelMember", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(&model.ChannelMember{}, (*model.AppError)(nil))

			if tc.apiCalls != nil {
				tc.apiCalls(api)
			}

			w := httptest.NewRecorder()
			request := httptest.NewRequest("DELETE", "/api/v2/subscriptions/channel/"+tc.subscriptionId, nil)
			if !tc.skipAuthorize {
				request.Header.Set("Mattermost-User-Id", model.NewId())
			}
			p.ServeHTTP(&mmplugin.Context{}, w, request)
			assert.Equal(t, tc.expectedStatusCode, w.Result().StatusCode)
		})
	}
}

func TestEditSubscription(t *testing.T) {
	for name, tc := range map[string]struct {
		subscription       string
		expectedStatusCode int
		skipAuthorize      bool
		apiCalls           func(*plugintest.API)
	}{
		"Invalid": {
			subscription:       "{}",
			expectedStatusCode: http.StatusBadRequest,
		},
		"Not Authorized": {
			subscription:       "{}",
			expectedStatusCode: http.StatusUnauthorized,
			skipAuthorize:      true,
		},
		"Won't Decode": {
			subscription:       "{woopsie",
			expectedStatusCode: http.StatusBadRequest,
		},
		"No channel id": {
			subscription:       `{"id": "aaaaaaaaaaaaaaaaaaaaaaaaab", "channel_id": "badchannelid", "fields": {}}`,
			expectedStatusCode: http.StatusBadRequest,
		},
		"No Id": {
			subscription:       `{"id": "badid", "channel_id": "aaaaaaaaaaaaaaaaaaaaaaaaab", "fields": {}}`,
			expectedStatusCode: http.StatusBadRequest,
		},
		"Editing subscription": {
			subscription:       `{"id": "aaaaaaaaaaaaaaaaaaaaaaaaab", "channel_id": "aaaaaaaaaaaaaaaaaaaaaaaaac", "filters": {"events": ["jira:issue_created"], "project": ["otherproject"]}}`,
			expectedStatusCode: http.StatusOK,
			apiCalls: checkHasSubscriptions([]jira.ChannelSubscription{
				jira.ChannelSubscription{
					Id:        "aaaaaaaaaaaaaaaaaaaaaaaaab",
					ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaac",
					Filters: map[string][]string{
						"events":  []string{"jira:issue_created"},
						"project": []string{"otherproject"},
					},
				},
			},
				withExistingChannelSubscriptions(
					[]jira.ChannelSubscription{
						jira.ChannelSubscription{
							Id:        "aaaaaaaaaaaaaaaaaaaaaaaaab",
							ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaac",
							Filters: map[string][]string{
								"events":  []string{"jira:issue_created"},
								"project": []string{"myproject"},
							},
						},
					}), t),
		},
	} {
		t.Run(name, func(t *testing.T) {
			api := &plugintest.API{}
			p := SetupTestPlugin(t, api, plugin.Config{
				MainConfig: plugin.MainConfig{
					EnableJiraUI:  false,
					WebhookSecret: "somesecret",
					BotUserName:   "someuser",
				},
			}, nil)

			api.On("GetChannelMember", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(&model.ChannelMember{}, (*model.AppError)(nil))

			if tc.apiCalls != nil {
				tc.apiCalls(api)
			}

			w := httptest.NewRecorder()
			request := httptest.NewRequest("PUT", "/api/v2/subscriptions/channel", ioutil.NopCloser(bytes.NewBufferString(tc.subscription)))
			if !tc.skipAuthorize {
				request.Header.Set("Mattermost-User-Id", model.NewId())
			}
			p.ServeHTTP(&mmplugin.Context{}, w, request)
			assert.Equal(t, tc.expectedStatusCode, w.Result().StatusCode)
		})
	}
}

func TestGetSubscriptionsForChannel(t *testing.T) {
	for name, tc := range map[string]struct {
		channelId             string
		expectedStatusCode    int
		skipAuthorize         bool
		apiCalls              func(*plugintest.API)
		returnedSubscriptions []jira.ChannelSubscription
	}{
		"Invalid": {
			channelId:          "nope",
			expectedStatusCode: http.StatusBadRequest,
		},
		"Not Authorized": {
			channelId:          model.NewId(),
			expectedStatusCode: http.StatusUnauthorized,
			skipAuthorize:      true,
		},
		"Only Subscription": {
			channelId:          "aaaaaaaaaaaaaaaaaaaaaaaaac",
			expectedStatusCode: http.StatusOK,
			returnedSubscriptions: []jira.ChannelSubscription{
				jira.ChannelSubscription{
					Id:        "aaaaaaaaaaaaaaaaaaaaaaaaab",
					ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaac",
					Filters: map[string][]string{
						"events":  []string{"jira:issue_created"},
						"project": []string{"myproject"},
					},
				},
			},
			apiCalls: hasSubscriptions(
				[]jira.ChannelSubscription{
					jira.ChannelSubscription{
						Id:        "aaaaaaaaaaaaaaaaaaaaaaaaab",
						ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaac",
						Filters: map[string][]string{
							"events":  []string{"jira:issue_created"},
							"project": []string{"myproject"},
						},
					},
				}, t),
		},
		"Multiple subscriptions": {
			channelId:          "aaaaaaaaaaaaaaaaaaaaaaaaac",
			expectedStatusCode: http.StatusOK,
			returnedSubscriptions: []jira.ChannelSubscription{
				jira.ChannelSubscription{
					Id:        "aaaaaaaaaaaaaaaaaaaaaaaaab",
					ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaac",
					Filters: map[string][]string{
						"events":  []string{"jira:issue_created"},
						"project": []string{"myproject"},
					},
				},
				jira.ChannelSubscription{
					Id:        "aaaaaaaaaaaaaaaaaaaaaaaaac",
					ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaac",
					Filters: map[string][]string{
						"events":  []string{"jira:issue_created"},
						"project": []string{"things"},
					},
				},
			},
			apiCalls: hasSubscriptions(
				[]jira.ChannelSubscription{
					jira.ChannelSubscription{
						Id:        "aaaaaaaaaaaaaaaaaaaaaaaaab",
						ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaac",
						Filters: map[string][]string{
							"events":  []string{"jira:issue_created"},
							"project": []string{"myproject"},
						},
					},
					jira.ChannelSubscription{
						Id:        "aaaaaaaaaaaaaaaaaaaaaaaaac",
						ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaac",
						Filters: map[string][]string{
							"events":  []string{"jira:issue_created"},
							"project": []string{"things"},
						},
					},
				}, t),
		},
		"Only in channel": {
			channelId:          "aaaaaaaaaaaaaaaaaaaaaaaaac",
			expectedStatusCode: http.StatusOK,
			returnedSubscriptions: []jira.ChannelSubscription{
				jira.ChannelSubscription{
					Id:        "aaaaaaaaaaaaaaaaaaaaaaaaab",
					ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaac",
					Filters: map[string][]string{
						"events":  []string{"jira:issue_created"},
						"project": []string{"myproject"},
					},
				},
			},
			apiCalls: hasSubscriptions(
				[]jira.ChannelSubscription{
					jira.ChannelSubscription{
						Id:        "aaaaaaaaaaaaaaaaaaaaaaaaab",
						ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaac",
						Filters: map[string][]string{
							"events":  []string{"jira:issue_created"},
							"project": []string{"myproject"},
						},
					},
					jira.ChannelSubscription{
						Id:        "aaaaaaaaaaaaaaaaaaaaaaaaac",
						ChannelId: "aaaaaaaaaaaaaaaaaaaaaaaaad",
						Filters: map[string][]string{
							"events":  []string{"jira:issue_created"},
							"project": []string{"things"},
						},
					},
				}, t),
		},
	} {
		t.Run(name, func(t *testing.T) {
			api := &plugintest.API{}
			p := SetupTestPlugin(t, api, plugin.Config{
				MainConfig: plugin.MainConfig{
					EnableJiraUI:  false,
					WebhookSecret: "somesecret",
					BotUserName:   "someuser",
				},
			}, nil)

			api.On("GetChannelMember", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(&model.ChannelMember{}, (*model.AppError)(nil))

			if tc.apiCalls != nil {
				tc.apiCalls(api)
			}

			w := httptest.NewRecorder()
			request := httptest.NewRequest("GET", "/api/v2/subscriptions/channel/"+tc.channelId, nil)
			if !tc.skipAuthorize {
				request.Header.Set("Mattermost-User-Id", model.NewId())
			}
			p.ServeHTTP(&mmplugin.Context{}, w, request)
			assert.Equal(t, tc.expectedStatusCode, w.Result().StatusCode)

			if tc.returnedSubscriptions != nil {
				subscriptions := []jira.ChannelSubscription{}
				body, _ := ioutil.ReadAll(w.Result().Body)
				err := json.NewDecoder(bytes.NewReader(body)).Decode(&subscriptions)
				assert.Nil(t, err)

				assert.Equal(t, tc.returnedSubscriptions, subscriptions)
			}
		})
	}
}
