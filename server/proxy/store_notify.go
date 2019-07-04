package proxy

import (
	"github.com/mattermost/mattermost-plugin-jira/server/upstream"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

const (
	WebsocketEventConnect        = "connect"
	WebsocketEventDisconnect     = "disconnect"
	WebsocketEventUpstreamStatus = "instance_status"
)

func StoreCurrentUpstreamNotify(api plugin.API, upstreamStore upstream.Store, up upstream.Upstream) error {
	err := upstreamStore.StoreCurrent(up)
	if err != nil {
		return err
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

func DeleteUpstreamNotify(api plugin.API, upstreamStore upstream.Store, upstreamKey string) error {
	err := upstreamStore.Delete(upstreamKey)
	if err != nil {
		return err
	}

	// Assume that this was the current instance, and notify the user
	api.PublishWebSocketEvent(
		WebsocketEventUpstreamStatus,
		map[string]interface{}{
			"instance_installed": false,
		},
		&model.WebsocketBroadcast{},
	)
	return nil
}

func StoreUserNotify(api plugin.API, up upstream.Upstream, u upstream.User) error {
	err := up.StoreUser(u)
	if err != nil {
		return err
	}

	api.PublishWebSocketEvent(
		WebsocketEventConnect,
		map[string]interface{}{
			"is_connected": true,
		},
		&model.WebsocketBroadcast{UserId: u.MattermostUserId()},
	)

	return nil
}

func DeleteUserNotify(api plugin.API, up upstream.Upstream, u upstream.User) error {
	err := up.DeleteUser(u)
	if err != nil {
		return err
	}

	api.PublishWebSocketEvent(
		WebsocketEventDisconnect,
		map[string]interface{}{
			"is_connected": false,
		},
		&model.WebsocketBroadcast{UserId: u.MattermostUserId()},
	)

	return nil
}
