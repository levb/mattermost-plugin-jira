package lib

import (

	// "github.com/andygrunwald/go-jira"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"

	"github.com/mattermost/mattermost-plugin-jira/server/upstream"
)

const (
	WebsocketEventConnect        = "connect"
	WebsocketEventDisconnect     = "disconnect"
	WebsocketEventUpstreamStatus = "instance_status"
)

func StoreCurrentUpstreamNotify(api plugin.API, upstreamStore upstream.Store, up upstream.Upstream) error {
	appErr := upstreamStore.StoreCurrent(up)
	if appErr != nil {
		return appErr
	}
	// Notify users we have installed an instance
	api.PublishWebSocketEvent(
		WebsocketEventUpstreamStatus,
		map[string]interface{}{
			"instance_installed": true,
		},
		&model.WebsocketBroadcast{},
	)
	return nil
}

func StoreUserNotify(api plugin.API,
	userStore upstream.UserStore,
	mattermostUserId string,
	user upstream.User) error {

	err := userStore.StoreUser(user)
	if err != nil {
		return err
	}

	api.PublishWebSocketEvent(
		WebsocketEventConnect,
		map[string]interface{}{
			"is_connected": true,
		},
		&model.WebsocketBroadcast{UserId: mattermostUserId},
	)

	return nil
}

func DeleteUserNotify(
	api plugin.API,
	userStore upstream.UserStore,
	mattermostUserId, userKey string,
) error {
	err := userStore.DeleteUser(mattermostUserId, userKey)
	if err != nil {
		return err
	}

	api.PublishWebSocketEvent(
		WebsocketEventDisconnect,
		map[string]interface{}{
			"is_connected": false,
		},
		&model.WebsocketBroadcast{UserId: mattermostUserId},
	)

	return nil
}
