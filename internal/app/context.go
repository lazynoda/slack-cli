package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"slack-cli/internal/config"
	"slack-cli/internal/output"
	slackapi "slack-cli/internal/slack"
)

type Settings struct {
	ConfigPath string
	Profile    string
	Timeout    time.Duration
	OutputMode output.Mode
}

type Context struct {
	Settings   Settings
	ConfigPath string
	Config     *config.File
	Output     *output.Printer

	slack            *slackapi.Client
	resolvedUsers    map[string]string
	resolvedChannels map[string]string
}

var slackIDPattern = regexp.MustCompile(`^[A-Z][A-Z0-9]+$`)

func NewContext(settings Settings, out io.Writer, errOut io.Writer) (*Context, error) {
	configPath, err := config.ResolvePath(settings.ConfigPath)
	if err != nil {
		return nil, err
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, err
	}

	return &Context{
		Settings:         settings,
		ConfigPath:       configPath,
		Config:           cfg,
		Output:           output.New(settings.OutputMode, out, errOut),
		resolvedUsers:    map[string]string{},
		resolvedChannels: map[string]string{},
	}, nil
}

func (c *Context) Token() (string, string) {
	if token := strings.TrimSpace(os.Getenv("SLACK_CLI_TOKEN")); token != "" {
		return token, "SLACK_CLI_TOKEN"
	}

	if token := strings.TrimSpace(os.Getenv("SLACK_TOKEN")); token != "" {
		return token, "SLACK_TOKEN"
	}

	profile := c.Config.Profile(c.Settings.Profile)
	if token := strings.TrimSpace(profile.Token); token != "" {
		return token, "config"
	}

	return "", ""
}

func (c *Context) Cookie() (string, string) {
	if cookie := strings.TrimSpace(os.Getenv("SLACK_CLI_COOKIE")); cookie != "" {
		return cookie, "SLACK_CLI_COOKIE"
	}

	if cookie := strings.TrimSpace(os.Getenv("SLACK_COOKIE")); cookie != "" {
		return cookie, "SLACK_COOKIE"
	}

	return "", ""
}

func (c *Context) RequireToken() (string, string, error) {
	token, source := c.Token()
	if token == "" {
		return "", "", errors.New("no Slack token configured; run `slack-cli auth set-token <token>` or set SLACK_CLI_TOKEN")
	}
	return token, source, nil
}

func (c *Context) SlackClient() (*slackapi.Client, error) {
	if c.slack != nil {
		return c.slack, nil
	}

	token, _, err := c.RequireToken()
	if err != nil {
		return nil, err
	}

	c.slack = slackapi.NewClient(slackapi.Options{
		Token:   token,
		Cookie:  firstValue(c.Cookie()),
		Timeout: c.Settings.Timeout,
	})
	return c.slack, nil
}

func (c *Context) SaveConfig() error {
	if err := config.Save(c.ConfigPath, c.Config); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	return nil
}

func (c *Context) RequestContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), c.Settings.Timeout)
}

func (c *Context) ResolveUser(ref string) (string, error) {
	trimmed := normalizeRef(ref)
	if trimmed == "" {
		return "", errors.New("user reference cannot be empty")
	}
	if slackIDPattern.MatchString(trimmed) && strings.HasPrefix(trimmed, "U") {
		return trimmed, nil
	}
	if cached := c.resolvedUsers[trimmed]; cached != "" {
		return cached, nil
	}

	client, err := c.SlackClient()
	if err != nil {
		return "", err
	}

	ctx, cancel := c.RequestContext()
	defer cancel()

	users, err := collectAllUsers(ctx, client)
	if err != nil {
		return "", err
	}

	matches := make([]slackapi.User, 0, 2)
	for _, user := range users {
		if userMatches(user, trimmed) {
			matches = append(matches, user)
		}
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("no Slack user found for %q", ref)
	}
	if len(matches) > 1 {
		names := make([]string, 0, len(matches))
		for _, match := range matches {
			names = append(names, fmt.Sprintf("@%s (%s)", match.Name, match.ID))
		}
		sort.Strings(names)
		return "", fmt.Errorf("multiple Slack users matched %q: %s", ref, strings.Join(names, ", "))
	}

	resolved := matches[0].ID
	c.rememberUser(trimmed, resolved, matches[0])
	return resolved, nil
}

func (c *Context) ResolveConversation(ref string) (string, error) {
	trimmed := normalizeRef(ref)
	if trimmed == "" {
		return "", errors.New("conversation reference cannot be empty")
	}
	if slackIDPattern.MatchString(trimmed) && isConversationID(trimmed) {
		return trimmed, nil
	}
	if cached := c.resolvedChannels[trimmed]; cached != "" {
		return cached, nil
	}

	client, err := c.SlackClient()
	if err != nil {
		return "", err
	}

	ctx, cancel := c.RequestContext()
	defer cancel()

	conversations, err := collectAllConversations(ctx, client)
	if err != nil {
		return "", err
	}

	matches := make([]slackapi.Conversation, 0, 2)
	for _, conversation := range conversations {
		if conversationMatches(conversation, trimmed) {
			matches = append(matches, conversation)
		}
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("no Slack conversation found for %q", ref)
	}
	if len(matches) > 1 {
		names := make([]string, 0, len(matches))
		for _, match := range matches {
			name := firstNonEmpty(match.Name, match.NameNormalized, match.User, match.ID)
			names = append(names, fmt.Sprintf("%s (%s)", name, match.ID))
		}
		sort.Strings(names)
		return "", fmt.Errorf("multiple Slack conversations matched %q: %s", ref, strings.Join(names, ", "))
	}

	resolved := matches[0].ID
	c.rememberConversation(trimmed, resolved, matches[0])
	return resolved, nil
}

func (c *Context) ResolveUsers(refs []string) ([]string, error) {
	resolved := make([]string, 0, len(refs))
	for _, ref := range refs {
		id, err := c.ResolveUser(ref)
		if err != nil {
			return nil, err
		}
		resolved = append(resolved, id)
	}
	return resolved, nil
}

func (c *Context) rememberUser(key string, id string, user slackapi.User) {
	keys := []string{
		key,
		normalizeRef(user.ID),
		normalizeRef(user.Name),
		normalizeRef("@" + user.Name),
		normalizeRef(user.RealName),
		normalizeRef(user.Profile.RealName),
		normalizeRef(user.Profile.DisplayName),
		normalizeRef("@" + user.Profile.DisplayName),
	}
	for _, k := range keys {
		if k != "" {
			c.resolvedUsers[k] = id
		}
	}
}

func (c *Context) rememberConversation(key string, id string, conversation slackapi.Conversation) {
	name := firstNonEmpty(conversation.Name, conversation.NameNormalized)
	keys := []string{
		key,
		normalizeRef(conversation.ID),
		normalizeRef(name),
		normalizeRef("#" + name),
	}
	for _, k := range keys {
		if k != "" {
			c.resolvedChannels[k] = id
		}
	}
}

func collectAllUsers(ctx context.Context, client *slackapi.Client) ([]slackapi.User, error) {
	var users []slackapi.User
	cursor := ""
	for {
		response, err := client.UsersList(ctx, 200, cursor)
		if err != nil {
			return nil, err
		}
		users = append(users, response.Members...)
		if response.ResponseMetadata.NextCursor == "" {
			return users, nil
		}
		cursor = response.ResponseMetadata.NextCursor
	}
}

func collectAllConversations(ctx context.Context, client *slackapi.Client) ([]slackapi.Conversation, error) {
	var conversations []slackapi.Conversation
	cursor := ""
	for {
		response, err := client.ConversationsList(ctx, 200, cursor, "public_channel,private_channel,im,mpim", false)
		if err != nil {
			return nil, err
		}
		conversations = append(conversations, response.Channels...)
		if response.ResponseMetadata.NextCursor == "" {
			return conversations, nil
		}
		cursor = response.ResponseMetadata.NextCursor
	}
}

func userMatches(user slackapi.User, ref string) bool {
	ref = normalizeRef(ref)
	candidates := []string{
		user.ID,
		user.Name,
		"@" + user.Name,
		user.RealName,
		user.Profile.RealName,
		user.Profile.DisplayName,
		"@" + user.Profile.DisplayName,
	}
	for _, candidate := range candidates {
		if normalizeRef(candidate) == ref && ref != "" {
			return true
		}
	}
	return false
}

func conversationMatches(conversation slackapi.Conversation, ref string) bool {
	ref = normalizeRef(ref)
	name := firstNonEmpty(conversation.Name, conversation.NameNormalized)
	candidates := []string{
		conversation.ID,
		name,
		"#" + name,
	}
	for _, candidate := range candidates {
		if normalizeRef(candidate) == ref && ref != "" {
			return true
		}
	}
	return false
}

func normalizeRef(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func isConversationID(value string) bool {
	return strings.HasPrefix(value, "C") || strings.HasPrefix(value, "G") || strings.HasPrefix(value, "D")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func firstValue(value string, _ string) string {
	return value
}
