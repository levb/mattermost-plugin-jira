package proxy

import (
	"github.com/mattermost/mattermost-plugin-jira/server/upstream"
	"github.com/mattermost/mattermost-server/model"
)

const (
	WebsocketEventUpstreamStatus = "instance_status"
)

func (p proxy) StoreCurrentUpstreamNotify(up upstream.Upstream) error {
	err := p.StoreCurrentUpstream(up)
	if err != nil {
		return err
	}
	// Notify users we have installed an instance
	p.context.API.PublishWebSocketEvent(
		WebsocketEventUpstreamStatus,
		map[string]interface{}{
			"instance_installed": true,
		},
		&model.WebsocketBroadcast{},
	)
	return nil
}

func (p proxy) DeleteUpstreamNotify(upstreamKey string) error {
	err := p.DeleteUpstream(upstreamKey)
	if err != nil {
		return err
	}

	// Assume that this was the current instance, and notify the user
	p.context.API.PublishWebSocketEvent(
		WebsocketEventUpstreamStatus,
		map[string]interface{}{
			"instance_installed": false,
		},
		&model.WebsocketBroadcast{},
	)
	return nil
}
