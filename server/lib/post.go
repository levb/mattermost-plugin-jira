package lib

import (
	"github.com/mattermost/mattermost-plugin-jira/server/upstream"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

func CreateBotDMPost(api plugin.API, up upstream.Upstream, botUserId string, mattermostUserId, message,
	postType string) (*model.Post, error) {
	u, err := up.LoadUser(mattermostUserId)
	if err != nil {
		return nil, err
	}
	if !u.Settings().Notifications {
		return nil, nil
	}

	channel, appErr := api.GetDirectChannel(mattermostUserId, botUserId)
	if appErr != nil {
		return nil, appErr
	}

	post := &model.Post{
		UserId:    botUserId,
		ChannelId: channel.Id,
		Message:   message,
		Type:      postType,
		Props: map[string]interface{}{
			"from_webhook": "true",
			// "override_username": PluginMattermostUsername,
			// "override_icon_url": PluginIconURL,
		},
	}

	_, appErr = api.CreatePost(post)
	if appErr != nil {
		return nil, appErr
	}

	return post, nil
}
