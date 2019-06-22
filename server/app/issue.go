// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package app

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/mattermost/mattermost-plugin-jira/server/store"

	"github.com/andygrunwald/go-jira"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-server/model"
	mmplugin "github.com/mattermost/mattermost-server/plugin"

	"github.com/mattermost/mattermost-plugin-jira/server/instance"
)

type CreateIssueRequest struct {
	PostId    string           `json:"post_id"`
	ChannelId string           `json:"channel_id"`
	Fields    jira.IssueFields `json:"fields"`
}

func CreateIssue(
	api mmplugin.API,
	siteURL string,
	jiraClient *jira.Client,
	instance instance.Instance,
	mattermostUserId string,
	req *CreateIssueRequest,
) (*jira.Issue, int, error) {

	fromPostId := req.PostId
	var fromPost *model.Post
	var appErr *model.AppError
	// If this issue is attached to a post, lets add a permalink to the post in the Jira Description
	if req.PostId != "" {
		fromPost, appErr = api.GetPost(fromPostId)
		if appErr != nil {
			return nil, http.StatusInternalServerError,
				errors.WithMessagef(appErr, "failed to load post %q", fromPostId)
		}
		if fromPost == nil {
			return nil, http.StatusInternalServerError,
				errors.Errorf("failed to load post %q: not found", fromPostId)
		}
		permalink := ""
		permalink, err := GetPermalink(api, siteURL, fromPostId, fromPost)
		if err != nil {
			return nil, http.StatusInternalServerError,
				errors.WithMessagef(err, "failed to get permalink for: %q", req.PostId)
		}

		if len(req.Fields.Description) > 0 {
			req.Fields.Description += "\n" + permalink
		} else {
			req.Fields.Description = permalink
		}
	}

	channelId := req.ChannelId
	if fromPost != nil {
		channelId = fromPost.ChannelId
	}

	createdIssue, resp, err := jiraClient.Issue.Create(&jira.Issue{
		Fields: &req.Fields,
	})
	if err != nil {
		message := "failed to create the issue, postId: " + req.PostId + ", channelId: " + channelId
		if resp != nil {
			bb, _ := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			message += ", details:" + string(bb)
		}
		return nil, http.StatusInternalServerError, errors.WithMessage(err, message)
	}

	// Upload file attachments in the background
	if fromPost != nil && len(fromPost.FileIds) > 0 {
		go func() {
			for _, fileId := range fromPost.FileIds {
				info, ae := api.GetFileInfo(fileId)
				if ae != nil {
					continue
				}
				// TODO: large file support? Ignoring errors for now is good enough...
				byteData, ae := api.ReadFile(info.Path)
				if ae != nil {
					// TODO report errors, as DMs from Jira bot?
					return
				}
				_, _, e := jiraClient.Issue.PostAttachment(createdIssue.ID, bytes.NewReader(byteData), info.Name)
				if e != nil {
					// TODO report errors, as DMs from Jira bot?
					return
				}
			}
		}()
	}

	rootId := req.PostId
	parentId := ""
	if fromPost.ParentId != "" {
		// the original post was a reply
		rootId = fromPost.RootId
		parentId = req.PostId
	}

	// Reply to the post with the issue link that was created
	replyPost := &model.Post{
		Message:   fmt.Sprintf("Created a Jira issue %v/browse/%v", instance.GetURL(), createdIssue.Key),
		ChannelId: channelId,
		RootId:    rootId,
		ParentId:  parentId,
		UserId:    mattermostUserId,
	}
	_, appErr = api.CreatePost(replyPost)
	if appErr != nil {
		return nil, http.StatusInternalServerError,
			errors.WithMessagef(appErr, "failed to create notification post: %q", req.PostId)
	}

	return createdIssue, 0, nil
}

type SearchIssueSummary struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

func GetSearchIssues(jiraClient *jira.Client, jqlString string) ([]SearchIssueSummary, int, error) {
	searchRes, resp, err := jiraClient.Issue.Search(jqlString, &jira.SearchOptions{
		MaxResults: 50,
		Fields:     []string{"key", "summary"},
	})
	if err != nil {
		message := "failed to get search results"
		if resp != nil {
			bb, _ := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			message += ", details: " + string(bb)
		}
		return nil, http.StatusInternalServerError, errors.WithMessage(err, message)
	}

	// We only need to send down a summary of the data
	resSummary := make([]SearchIssueSummary, 0, len(searchRes))
	for _, res := range searchRes {
		resSummary = append(resSummary, SearchIssueSummary{
			Value: res.Key,
			Label: res.Key + ": " + res.Fields.Summary,
		})
	}

	return resSummary, 0, nil
}

type AttachCommentToIssueRequest struct {
	PostId   string `json:"post_id"`
	IssueKey string `json:"issueKey"`
}

func AttachCommentToIssue(api mmplugin.API, siteURL string, jiraClient *jira.Client,
	instance instance.Instance, mattermostUserId string, req AttachCommentToIssueRequest,
	user store.User) (*jira.Comment, int, error) {

	// Add a permalink to the post to the issue description
	post, appErr := api.GetPost(req.PostId)
	if appErr != nil || post == nil {
		return nil, http.StatusInternalServerError,
			errors.WithMessagef(appErr, "failed to load or find post %q", req.PostId)
	}

	commentUser, appErr := api.GetUser(post.UserId)
	if appErr != nil {
		return nil, http.StatusInternalServerError,
			errors.WithMessagef(appErr, "failed to load User %q", post.UserId)
	}

	permalink, err := GetPermalink(api, siteURL, req.PostId, post)
	if err != nil {
		return nil, http.StatusInternalServerError,
			errors.WithMessagef(err, "failed to get permalink for %q", req.PostId)
	}

	permalinkMessage := fmt.Sprintf("*@%s attached a* [message|%s] *from @%s*\n",
		user.User.Name, permalink, commentUser.Username)

	var jiraComment jira.Comment
	jiraComment.Body = permalinkMessage
	jiraComment.Body += post.Message

	commentAdded, _, err := jiraClient.Issue.AddComment(req.IssueKey, &jiraComment)
	if err != nil {
		return nil, http.StatusInternalServerError,
			errors.WithMessagef(err, "failed to attach the comment, postId: %q", req.PostId)
	}

	rootId := req.PostId
	parentId := ""
	if post.ParentId != "" {
		// the original post was a reply
		rootId = post.RootId
		parentId = req.PostId
	}

	// Reply to the post with the issue link that was created
	reply := &model.Post{
		Message: fmt.Sprintf("Message attached to [%v](%v/browse/%v)",
			req.IssueKey, instance.GetURL(), req.IssueKey),
		ChannelId: post.ChannelId,
		RootId:    rootId,
		ParentId:  parentId,
		UserId:    mattermostUserId,
	}
	_, appErr = api.CreatePost(reply)
	if appErr != nil {
		return nil, http.StatusInternalServerError,
			errors.WithMessagef(appErr, "failed to create notification post %q", req.PostId)
	}

	return commentAdded, 0, nil
}

func getCreateIssueMetadata(jiraClient *jira.Client) (*jira.CreateMetaInfo, error) {
	cimd, resp, err := jiraClient.Issue.GetCreateMetaWithOptions(&jira.GetQueryOptions{
		Expand: "projects.issuetypes.fields",
	})
	if err != nil {
		message := "failed to get CreateIssue metadata"
		if resp != nil {
			bb, _ := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			message += ", details:" + string(bb)
		}
		return nil, errors.WithMessage(err, message)
	}
	return cimd, nil
}

func GetPermalink(api mmplugin.API, siteURL, postId string, post *model.Post) (string, error) {
	channel, appErr := api.GetChannel(post.ChannelId)
	if appErr != nil {
		return "", errors.WithMessage(appErr, "failed to get ChannelId, ChannelId: "+post.ChannelId)
	}

	team, appErr := api.GetTeam(channel.TeamId)
	if appErr != nil {
		return "", errors.WithMessage(appErr, "failed to get team, TeamId: "+channel.TeamId)
	}

	permalink := fmt.Sprintf("%v/%v/pl/%v", siteURL, team.Name, postId)
	return permalink, nil
}

func TransitionIssue(jiraClient *jira.Client, instance instance.Instance, issueKey, toState string) (string, error) {
	transitions, _, err := jiraClient.Issue.GetTransitions(issueKey)
	if err != nil {
		return "", errors.New("We couldn't find the issue key. Please confirm the issue key and try again. You may not have permissions to access this issue.")
	}
	if len(transitions) < 1 {
		return "", errors.New("You do not have the appropriate permissions to perform this action. Please contact your Jira administrator.")
	}

	transitionToUse := jira.Transition{}
	matchingStates := []string{}
	availableStates := []string{}
	for _, transition := range transitions {
		if strings.Contains(strings.ToLower(transition.To.Name), strings.ToLower(toState)) {
			matchingStates = append(matchingStates, transition.To.Name)
			transitionToUse = transition
		}
		availableStates = append(availableStates, transition.To.Name)
	}

	switch len(matchingStates) {
	case 0:
		return "", errors.Errorf("%q is not a valid state. Please use one of: %q",
			toState, strings.Join(availableStates, ", "))

	case 1:
		// proceed

	default:
		return "", errors.Errorf("please be more specific, %q matched several states: %q",
			toState, strings.Join(matchingStates, ", "))
	}

	if _, err := jiraClient.Issue.DoTransition(issueKey, transitionToUse.ID); err != nil {
		return "", err
	}

	msg := fmt.Sprintf("[%s](%v/browse/%v) transitioned to `%s`", issueKey, instance.GetURL(), issueKey, transitionToUse.To.Name)
	return msg, nil
}
