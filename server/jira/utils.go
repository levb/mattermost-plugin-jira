package jira

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/url"
	"path"
	"regexp"
	"strings"

	// "github.com/andygrunwald/go-jira"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"

	"github.com/mattermost/mattermost-plugin-jira/server/upstream"
	"github.com/mattermost/mattermost-plugin-jira/server/store"
	"github.com/mattermost/mattermost-plugin-jira/server/upstream"
)

const WebsocketEventUpstreamStatus = "instance_status"

const StoreKeyTokenSecret = "token_secret"
const StoreKeyRSAPrivateKey = "rsa_key"

func normalizeInstallURL(jiraURL string) (string, error) {
	u, err := url.Parse(jiraURL)
	if err != nil {
		return "", err
	}
	if u.Host == "" {
		ss := strings.Split(u.Path, "/")
		if len(ss) > 0 && ss[0] != "" {
			u.Host = ss[0]
			u.Path = path.Join(ss[1:]...)
		}
		u, err = url.Parse(u.String())
		if err != nil {
			return "", err
		}
	}
	if u.Host == "" {
		return "", errors.Errorf("Invalid URL, no hostname: %q", jiraURL)
	}
	if u.Scheme == "" {
		u.Scheme = "https"
	}
	return strings.TrimSuffix(u.String(), "/"), nil
}

func CreateBotDMPost(api plugin.API, botUserId, mattermostUserId string, user user.User, message, 
	postType string) (post *model.Post, returnErr error) {

	if !user.Settings().Notifications {
		return nil, nil
	}

	channel, appErr := api.GetDirectChannel(userId, botUserId)
	if appErr != nil {
		return nil, appErr
	}

	post = &model.Post{
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

func StoreCurrentUpstreamAndNotify(api plugin.API,
	currentUpstreamStore instance.CurrentUpstreamStore,
	instance instance.Upstream) error {

	appErr := currentUpstreamStore.Store(instance)
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

func parseJiraUsernamesFromText(text string) []string {
	usernameMap := map[string]bool{}
	usernames := []string{}

	var re = regexp.MustCompile(`(?m)\[~([a-zA-Z0-9-_.\+]+)\]`)
	for _, match := range re.FindAllString(text, -1) {
		name := match[:len(match)-1]
		name = name[2:]
		if !usernameMap[name] {
			usernames = append(usernames, name)
			usernameMap[name] = true
		}
	}

	return usernames
}

func EnsureRSAPrivateKey(s store.Store) (*rsa.PrivateKey, error) {
	// Ensure we generate the secrets on first start
	rsaPrivateKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to generate private key")
	}
	rsaPrivateKeyBytes, err := json.Marshal(rsaPrivateKey)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to marshal private key")
	}
	newRSAPrivateKeyBytes, err := s.Ensure(StoreKeyRSAPrivateKey, rsaPrivateKeyBytes)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to generate private key")
	}
	newRSAPrivateKey := &rsa.PrivateKey{}
	err = json.Unmarshal(newRSAPrivateKeyBytes, newRSAPrivateKey)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to unmarshal private key")
	}
	return rsaPrivateKey, nil
}

func EnsureAuthTokenSecret(s store.Store) ([]byte, error) {
	// Ensure we generate the secrets on first start
	secret := make([]byte, 32)
	_, err := rand.Reader.Read(secret)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to generate token secret")
	}
	return s.Ensure(StoreKeyTokenSecret, secret)
}

// func parseJiraIssuesFromText(text string, keys []string) []string {
// 	issueMap := map[string]bool{}
// 	issues := []string{}

// 	for _, key := range keys {
// 		var re = regexp.MustCompile(fmt.Sprintf(`(?m)%s-[0-9]+`, key))
// 		for _, match := range re.FindAllString(text, -1) {
// 			if !issueMap[match] {
// 				issues = append(issues, match)
// 				issueMap[match] = true
// 			}
// 		}
// 	}

// 	return issues
// }
