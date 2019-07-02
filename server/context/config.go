// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package context

// Config is the main plugin configuration, stored in the Mattermost config,
// and updated via Mattermost system console, CLI, or other means
type Config struct {
	// Setting to turn on/off the webapp components of this plugin
	EnableJiraUI bool `json:"enablejiraui"`

	// Bot username
	BotUserName string `json:"username"`

	// Legacy 1.x Webhook secret
	WebhookSecret string `json:"secret"`
}
