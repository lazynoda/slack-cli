package cli

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"slack-cli/internal/app"
	"slack-cli/internal/output"
	slackapi "slack-cli/internal/slack"
)

type CLI struct {
	Globals       Globals          `embed:""`
	Auth          AuthCmd          `cmd:"" help:"Authentication commands."`
	API           APICmd           `cmd:"" help:"Call raw Slack Web API methods."`
	Resolve       ResolveCmd       `cmd:"" help:"Resolve friendly references to Slack IDs."`
	Users         UsersCmd         `cmd:"" help:"User commands."`
	Conversations ConversationsCmd `cmd:"" help:"Conversation commands built on conversations.* methods."`
	Chat          ChatCmd          `cmd:"" help:"Message commands built on chat.* methods."`
	Search        SearchCmd        `cmd:"" help:"Search commands."`
}

type Globals struct {
	Config  string        `help:"Config file path."`
	Profile string        `default:"default" help:"Config profile name."`
	Timeout time.Duration `default:"10s" help:"Request timeout."`
	JSON    bool          `help:"Output JSON."`
	Plain   bool          `help:"Output plain text for scripting."`
}

type AuthCmd struct {
	Status   AuthStatusCmd   `cmd:"" help:"Show configured authentication state."`
	SetToken AuthSetTokenCmd `cmd:"" name:"set-token" help:"Save a Slack token in the current profile."`
	Clear    AuthClearCmd    `cmd:"" help:"Clear the stored token for the current profile."`
	Test     AuthTestCmd     `cmd:"" help:"Call auth.test with the active token."`
}

type APICmd struct {
	Call APICallCmd `cmd:"" help:"Call any Slack Web API method."`
}

type ResolveCmd struct {
	User         ResolveUserCmd         `cmd:"" help:"Resolve a user reference to a Slack user."`
	Conversation ResolveConversationCmd `cmd:"" help:"Resolve a conversation reference to a Slack conversation."`
}

type APICallCmd struct {
	Method     string   `arg:"" help:"Slack Web API method name, for example conversations.list."`
	HTTPMethod string   `name:"http" default:"POST" enum:"GET,POST" help:"HTTP verb to use."`
	Params     []string `name:"param" help:"Request parameter in key=value form. Repeatable."`
}

type ResolveUserCmd struct {
	User string `arg:"" help:"User reference: ID, @name, display name, or real name."`
}

type ResolveConversationCmd struct {
	Channel string `arg:"" help:"Conversation reference: ID, #name, or name."`
}

type UsersCmd struct {
	List     UsersListCmd     `cmd:"" help:"List workspace users."`
	Info     UsersInfoCmd     `cmd:"" help:"Show one user."`
	Presence UsersPresenceCmd `cmd:"" help:"Show one user's presence."`
}

type UsersListCmd struct {
	Limit  int    `default:"100" help:"Maximum users per page."`
	Cursor string `help:"Pagination cursor."`
	All    bool   `help:"Follow response_metadata.next_cursor until exhausted."`
}

type UsersInfoCmd struct {
	User string `arg:"" help:"User reference: ID, @name, display name, or real name."`
}

type UsersPresenceCmd struct {
	User string `arg:"" help:"User reference: ID, @name, display name, or real name."`
}

type ConversationsCmd struct {
	List      ConversationsListCmd      `cmd:"" help:"List conversations."`
	Info      ConversationsInfoCmd      `cmd:"" help:"Show one conversation."`
	History   ConversationsHistoryCmd   `cmd:"" help:"Show conversation history."`
	Replies   ConversationsRepliesCmd   `cmd:"" help:"Show thread replies."`
	Members   ConversationsMembersCmd   `cmd:"" help:"List conversation members."`
	Open      ConversationsOpenCmd      `cmd:"" help:"Open or resume a DM or MPDM."`
	Join      ConversationsJoinCmd      `cmd:"" help:"Join a conversation."`
	Leave     ConversationsLeaveCmd     `cmd:"" help:"Leave a conversation."`
	Create    ConversationsCreateCmd    `cmd:"" help:"Create a conversation."`
	Rename    ConversationsRenameCmd    `cmd:"" help:"Rename a conversation."`
	Topic     ConversationsTopicCmd     `cmd:"" help:"Set a conversation topic."`
	Purpose   ConversationsPurposeCmd   `cmd:"" help:"Set a conversation purpose."`
	Archive   ConversationsArchiveCmd   `cmd:"" help:"Archive a conversation."`
	Unarchive ConversationsUnarchiveCmd `cmd:"" help:"Unarchive a conversation."`
	Mark      ConversationsMarkCmd      `cmd:"" help:"Set the read cursor for a conversation."`
	Unread    ConversationsUnreadCmd    `cmd:"" help:"Show messages newer than the conversation's last_read cursor when available."`
}

type ConversationsListCmd struct {
	Limit           int    `default:"100" help:"Maximum conversations per page."`
	Cursor          string `help:"Pagination cursor."`
	All             bool   `help:"Follow response_metadata.next_cursor until exhausted."`
	Types           string `default:"public_channel,private_channel,im,mpim" help:"Conversation types passed to conversations.list."`
	ExcludeArchived bool   `help:"Exclude archived conversations."`
}

type ConversationsInfoCmd struct {
	Channel           string `arg:"" help:"Conversation reference: ID, #name, or name."`
	IncludeNumMembers bool   `help:"Request num_members from conversations.info."`
}

type ConversationsHistoryCmd struct {
	Channel   string `arg:"" help:"Conversation reference: ID, #name, or name."`
	Limit     int    `default:"20" help:"Maximum messages per page."`
	Cursor    string `help:"Pagination cursor."`
	All       bool   `help:"Follow response_metadata.next_cursor until exhausted."`
	Oldest    string `help:"Oldest message timestamp to include after."`
	Latest    string `help:"Latest message timestamp to include before."`
	Inclusive bool   `help:"Include messages equal to oldest/latest boundaries."`
}

type ConversationsRepliesCmd struct {
	Channel   string `arg:"" help:"Conversation reference: ID, #name, or name."`
	TS        string `arg:"" help:"Parent message timestamp."`
	Limit     int    `default:"20" help:"Maximum replies per page."`
	Cursor    string `help:"Pagination cursor."`
	All       bool   `help:"Follow response_metadata.next_cursor until exhausted."`
	Oldest    string `help:"Oldest reply timestamp to include after."`
	Latest    string `help:"Latest reply timestamp to include before."`
	Inclusive bool   `help:"Include replies equal to oldest/latest boundaries."`
}

type ConversationsMembersCmd struct {
	Channel string `arg:"" help:"Conversation reference: ID, #name, or name."`
	Limit   int    `default:"100" help:"Maximum members per page."`
	Cursor  string `help:"Pagination cursor."`
	All     bool   `help:"Follow response_metadata.next_cursor until exhausted."`
}

type ConversationsOpenCmd struct {
	Users []string `arg:"" help:"One or more user references. Pass multiple users to open an MPDM."`
}

type ConversationsJoinCmd struct {
	Channel string `arg:"" help:"Conversation reference: ID, #name, or name."`
}

type ConversationsLeaveCmd struct {
	Channel string `arg:"" help:"Conversation reference: ID, #name, or name."`
}

type ConversationsCreateCmd struct {
	Name    string `arg:"" help:"Conversation name."`
	Private bool   `help:"Create a private conversation."`
}

type ConversationsRenameCmd struct {
	Channel string `arg:"" help:"Conversation reference: ID, #name, or name."`
	Name    string `arg:"" help:"New conversation name."`
}

type ConversationsTopicCmd struct {
	Channel string `arg:"" help:"Conversation reference: ID, #name, or name."`
	Topic   string `arg:"" help:"New topic."`
}

type ConversationsPurposeCmd struct {
	Channel string `arg:"" help:"Conversation reference: ID, #name, or name."`
	Purpose string `arg:"" help:"New purpose."`
}

type ConversationsArchiveCmd struct {
	Channel string `arg:"" help:"Conversation reference: ID, #name, or name."`
}

type ConversationsUnarchiveCmd struct {
	Channel string `arg:"" help:"Conversation reference: ID, #name, or name."`
}

type ConversationsMarkCmd struct {
	Channel string `arg:"" help:"Conversation reference: ID, #name, or name."`
	TS      string `arg:"" help:"Message timestamp to mark as read."`
}

type ConversationsUnreadCmd struct {
	Channel string `arg:"" help:"Conversation reference: ID, #name, or name."`
	Limit   int    `default:"50" help:"Maximum unread messages per page."`
	All     bool   `help:"Follow response_metadata.next_cursor until exhausted."`
}

type ChatCmd struct {
	Send      ChatSendCmd      `cmd:"" help:"Send a message with chat.postMessage."`
	DM        ChatDMCmd        `cmd:"" help:"Open a DM or MPDM and send a message."`
	Ephemeral ChatEphemeralCmd `cmd:"" help:"Send an ephemeral message."`
	Update    ChatUpdateCmd    `cmd:"" help:"Update a message."`
	Delete    ChatDeleteCmd    `cmd:"" help:"Delete a message."`
	Permalink ChatPermalinkCmd `cmd:"" help:"Fetch a message permalink."`
}

type ChatSendCmd struct {
	Channel  string `arg:"" help:"Conversation reference: ID, #name, or name."`
	Text     string `arg:"" help:"Message text."`
	ThreadTS string `name:"thread-ts" help:"Reply in an existing thread."`
}

type ChatDMCmd struct {
	Users []string `arg:"" help:"One or more user references followed by the message text as the final argument."`
}

type ChatEphemeralCmd struct {
	Channel  string `arg:"" help:"Conversation reference: ID, #name, or name."`
	User     string `arg:"" help:"User reference receiving the ephemeral message."`
	Text     string `arg:"" help:"Message text."`
	ThreadTS string `name:"thread-ts" help:"Reply in an existing thread."`
}

type ChatUpdateCmd struct {
	Channel string `arg:"" help:"Conversation reference: ID, #name, or name."`
	TS      string `arg:"" help:"Message timestamp."`
	Text    string `arg:"" help:"Updated message text."`
}

type ChatDeleteCmd struct {
	Channel string `arg:"" help:"Conversation reference: ID, #name, or name."`
	TS      string `arg:"" help:"Message timestamp."`
}

type ChatPermalinkCmd struct {
	Channel string `arg:"" help:"Conversation reference: ID, #name, or name."`
	TS      string `arg:"" help:"Message timestamp."`
}

type SearchCmd struct {
	Messages SearchMessagesCmd `cmd:"" help:"Search messages with search.messages."`
}

type SearchMessagesCmd struct {
	Query     string `arg:"" help:"Search query."`
	Count     int    `default:"20" help:"Results per page."`
	Page      int    `default:"1" help:"Page number."`
	Cursor    string `help:"Search cursor for cursormark pagination."`
	Sort      string `default:"score" enum:"score,timestamp" help:"Sort order."`
	SortDir   string `name:"sort-dir" default:"desc" enum:"asc,desc" help:"Sort direction."`
	Highlight bool   `help:"Enable Slack highlight markers."`
}

func New() *CLI {
	return &CLI{}
}

func (g *Globals) Settings() (app.Settings, error) {
	if g.JSON && g.Plain {
		return app.Settings{}, errors.New("--json and --plain cannot be combined")
	}

	mode := output.Human
	if g.JSON {
		mode = output.JSON
	} else if g.Plain {
		mode = output.Plain
	}

	return app.Settings{
		ConfigPath: g.Config,
		Profile:    g.Profile,
		Timeout:    g.Timeout,
		OutputMode: mode,
	}, nil
}

func (c *AuthStatusCmd) Run(appCtx *app.Context) error {
	token, source := appCtx.Token()
	cookie, cookieSource := appCtx.Cookie()
	response := map[string]any{
		"config_path":       appCtx.ConfigPath,
		"profile":           appCtx.Settings.Profile,
		"token_source":      source,
		"configured":        token != "",
		"token_masked":      maskToken(token),
		"cookie_source":     cookieSource,
		"cookie_configured": cookie != "",
	}

	switch appCtx.Output.Mode() {
	case output.JSON:
		return appCtx.Output.JSON(response)
	case output.Plain:
		appCtx.Output.Lines([]string{
			"config_path=" + appCtx.ConfigPath,
			"profile=" + appCtx.Settings.Profile,
			"configured=" + strconv.FormatBool(token != ""),
			"token_source=" + source,
			"token_masked=" + maskToken(token),
			"cookie_configured=" + strconv.FormatBool(cookie != ""),
			"cookie_source=" + cookieSource,
		})
	default:
		appCtx.Output.Lines([]string{
			"Config: " + appCtx.ConfigPath,
			"Profile: " + appCtx.Settings.Profile,
			"Configured: " + strconv.FormatBool(token != ""),
			"Token source: " + valueOr(source, "not configured"),
			"Token: " + valueOr(maskToken(token), "not configured"),
			"Cookie configured: " + strconv.FormatBool(cookie != ""),
			"Cookie source: " + valueOr(cookieSource, "not configured"),
		})
	}

	return nil
}

type AuthStatusCmd struct{}

func (c *AuthSetTokenCmd) Run(appCtx *app.Context) error {
	token := strings.TrimSpace(c.Token)
	if token == "" {
		return errors.New("token cannot be empty")
	}

	appCtx.Config.SetToken(appCtx.Settings.Profile, token)
	if err := appCtx.SaveConfig(); err != nil {
		return err
	}

	response := map[string]any{
		"config_path":  appCtx.ConfigPath,
		"profile":      appCtx.Settings.Profile,
		"token_masked": maskToken(token),
	}

	if appCtx.Output.Mode() == output.JSON {
		return appCtx.Output.JSON(response)
	}

	if appCtx.Output.Mode() == output.Plain {
		appCtx.Output.Lines([]string{
			"config_path=" + appCtx.ConfigPath,
			"profile=" + appCtx.Settings.Profile,
			"token_masked=" + maskToken(token),
		})
		return nil
	}

	appCtx.Output.Lines([]string{
		"Saved token for profile " + appCtx.Settings.Profile,
		"Config: " + appCtx.ConfigPath,
		"Token: " + maskToken(token),
	})
	return nil
}

type AuthSetTokenCmd struct {
	Token string `arg:"" help:"Slack token to store, for example xoxb-... or xoxp-..."`
}

func (c *AuthClearCmd) Run(appCtx *app.Context) error {
	appCtx.Config.ClearToken(appCtx.Settings.Profile)
	if err := appCtx.SaveConfig(); err != nil {
		return err
	}

	response := map[string]any{
		"config_path": appCtx.ConfigPath,
		"profile":     appCtx.Settings.Profile,
		"cleared":     true,
	}

	if appCtx.Output.Mode() == output.JSON {
		return appCtx.Output.JSON(response)
	}

	if appCtx.Output.Mode() == output.Plain {
		appCtx.Output.Lines([]string{
			"config_path=" + appCtx.ConfigPath,
			"profile=" + appCtx.Settings.Profile,
			"cleared=true",
		})
		return nil
	}

	appCtx.Output.Lines([]string{
		"Cleared stored token for profile " + appCtx.Settings.Profile,
		"Config: " + appCtx.ConfigPath,
	})
	return nil
}

type AuthClearCmd struct{}

func (c *AuthTestCmd) Run(appCtx *app.Context) error {
	client, err := appCtx.SlackClient()
	if err != nil {
		return err
	}

	ctx, cancel := appCtx.RequestContext()
	defer cancel()

	response, err := client.AuthTest(ctx)
	if err != nil {
		return err
	}

	switch appCtx.Output.Mode() {
	case output.JSON:
		return appCtx.Output.JSON(response)
	case output.Plain:
		appCtx.Output.Lines([]string{
			"url=" + response.URL,
			"team=" + response.Team,
			"team_id=" + response.TeamID,
			"user=" + response.User,
			"user_id=" + response.UserID,
			"bot_id=" + response.BotID,
			"scope=" + response.Scope,
		})
	default:
		appCtx.Output.Lines([]string{
			"URL: " + response.URL,
			"Team: " + response.Team + " (" + response.TeamID + ")",
			"User: " + response.User + " (" + response.UserID + ")",
			"Bot ID: " + valueOr(response.BotID, "n/a"),
			"Scope: " + valueOr(response.Scope, "n/a"),
		})
	}

	return nil
}

type AuthTestCmd struct{}

func (c *APICallCmd) Run(appCtx *app.Context) error {
	params, err := parseParams(c.Params)
	if err != nil {
		return err
	}

	client, err := appCtx.SlackClient()
	if err != nil {
		return err
	}

	ctx, cancel := appCtx.RequestContext()
	defer cancel()

	var response map[string]any
	if err := client.CallWithHTTPMethod(ctx, c.Method, c.HTTPMethod, params, &response); err != nil {
		return err
	}

	return appCtx.Output.JSON(response)
}

func (c *ResolveUserCmd) Run(appCtx *app.Context) error {
	userID, err := appCtx.ResolveUser(c.User)
	if err != nil {
		return err
	}

	client, err := appCtx.SlackClient()
	if err != nil {
		return err
	}

	ctx, cancel := appCtx.RequestContext()
	defer cancel()

	response, err := client.UsersInfo(ctx, userID)
	if err != nil {
		return err
	}

	if appCtx.Output.Mode() == output.JSON {
		return appCtx.Output.JSON(map[string]any{
			"input": c.User,
			"id":    userID,
			"user":  response.User,
		})
	}

	if appCtx.Output.Mode() == output.Plain {
		appCtx.Output.Lines([]string{
			"input=" + c.User,
			"id=" + userID,
			"name=" + response.User.Name,
			"real_name=" + firstNonEmpty(response.User.RealName, response.User.Profile.RealName),
		})
		return nil
	}

	appCtx.Output.Lines([]string{
		"Input: " + c.User,
		"ID: " + userID,
		"Name: " + response.User.Name,
		"Real name: " + valueOr(firstNonEmpty(response.User.RealName, response.User.Profile.RealName), "n/a"),
	})
	return nil
}

func (c *ResolveConversationCmd) Run(appCtx *app.Context) error {
	channelID, err := appCtx.ResolveConversation(c.Channel)
	if err != nil {
		return err
	}

	client, err := appCtx.SlackClient()
	if err != nil {
		return err
	}

	ctx, cancel := appCtx.RequestContext()
	defer cancel()

	response, err := client.ConversationsInfo(ctx, channelID, true)
	if err != nil {
		return err
	}

	name := firstNonEmpty(response.Channel.Name, response.Channel.NameNormalized)
	if appCtx.Output.Mode() == output.JSON {
		return appCtx.Output.JSON(map[string]any{
			"input":        c.Channel,
			"id":           channelID,
			"conversation": response.Channel,
		})
	}

	if appCtx.Output.Mode() == output.Plain {
		appCtx.Output.Lines([]string{
			"input=" + c.Channel,
			"id=" + channelID,
			"name=" + name,
			"type=" + conversationType(response.Channel),
		})
		return nil
	}

	appCtx.Output.Lines([]string{
		"Input: " + c.Channel,
		"ID: " + channelID,
		"Name: " + valueOr(name, "n/a"),
		"Type: " + conversationType(response.Channel),
	})
	return nil
}

func (c *UsersListCmd) Run(appCtx *app.Context) error {
	if c.Limit <= 0 {
		return errors.New("--limit must be greater than zero")
	}

	client, err := appCtx.SlackClient()
	if err != nil {
		return err
	}

	ctx, cancel := appCtx.RequestContext()
	defer cancel()

	users, err := collectUsers(ctx, client, c.Limit, c.Cursor, c.All)
	if err != nil {
		return err
	}

	switch appCtx.Output.Mode() {
	case output.JSON:
		return appCtx.Output.JSON(users)
	case output.Plain:
		return appCtx.Output.Table(nil, userRows(users))
	default:
		return appCtx.Output.Table([]string{"ID", "Name", "Real Name", "Email", "Deleted", "Bot"}, userRows(users))
	}
}

func (c *UsersInfoCmd) Run(appCtx *app.Context) error {
	client, err := appCtx.SlackClient()
	if err != nil {
		return err
	}

	ctx, cancel := appCtx.RequestContext()
	defer cancel()

	userID, err := appCtx.ResolveUser(c.User)
	if err != nil {
		return err
	}

	response, err := client.UsersInfo(ctx, userID)
	if err != nil {
		return err
	}

	return printUser(appCtx, response.User)
}

func (c *UsersPresenceCmd) Run(appCtx *app.Context) error {
	client, err := appCtx.SlackClient()
	if err != nil {
		return err
	}

	ctx, cancel := appCtx.RequestContext()
	defer cancel()

	userID, err := appCtx.ResolveUser(c.User)
	if err != nil {
		return err
	}

	response, err := client.UsersGetPresence(ctx, userID)
	if err != nil {
		return err
	}

	switch appCtx.Output.Mode() {
	case output.JSON:
		return appCtx.Output.JSON(response)
	case output.Plain:
		appCtx.Output.Lines([]string{
			"presence=" + response.Presence,
			"online=" + strconv.FormatBool(response.Online),
			"auto_away=" + strconv.FormatBool(response.AutoAway),
			"manual_away=" + strconv.FormatBool(response.ManualAway),
			"connection_count=" + strconv.Itoa(response.ConnectionCount),
			"last_activity=" + strconv.FormatInt(response.LastActivity, 10),
		})
	default:
		appCtx.Output.Lines([]string{
			"Presence: " + response.Presence,
			"Online: " + strconv.FormatBool(response.Online),
			"Auto away: " + strconv.FormatBool(response.AutoAway),
			"Manual away: " + strconv.FormatBool(response.ManualAway),
			"Connection count: " + strconv.Itoa(response.ConnectionCount),
			"Last activity: " + strconv.FormatInt(response.LastActivity, 10),
		})
	}

	return nil
}

func (c *ConversationsListCmd) Run(appCtx *app.Context) error {
	if c.Limit <= 0 {
		return errors.New("--limit must be greater than zero")
	}

	client, err := appCtx.SlackClient()
	if err != nil {
		return err
	}

	ctx, cancel := appCtx.RequestContext()
	defer cancel()

	channels, err := collectConversations(ctx, client, c.Limit, c.Cursor, c.All, c.Types, c.ExcludeArchived)
	if err != nil {
		return err
	}

	switch appCtx.Output.Mode() {
	case output.JSON:
		return appCtx.Output.JSON(channels)
	case output.Plain:
		return appCtx.Output.Table(nil, conversationRows(channels))
	default:
		return appCtx.Output.Table([]string{"ID", "Name", "Type", "Archived", "Members", "Unread", "Topic"}, conversationRows(channels))
	}
}

func (c *ConversationsInfoCmd) Run(appCtx *app.Context) error {
	client, err := appCtx.SlackClient()
	if err != nil {
		return err
	}

	ctx, cancel := appCtx.RequestContext()
	defer cancel()

	channelID, err := appCtx.ResolveConversation(c.Channel)
	if err != nil {
		return err
	}

	response, err := client.ConversationsInfo(ctx, channelID, c.IncludeNumMembers)
	if err != nil {
		return err
	}

	return printConversation(appCtx, response.Channel)
}

func (c *ConversationsHistoryCmd) Run(appCtx *app.Context) error {
	if c.Limit <= 0 {
		return errors.New("--limit must be greater than zero")
	}

	client, err := appCtx.SlackClient()
	if err != nil {
		return err
	}

	ctx, cancel := appCtx.RequestContext()
	defer cancel()

	channelID, err := appCtx.ResolveConversation(c.Channel)
	if err != nil {
		return err
	}

	messages, err := collectHistory(ctx, client, channelID, c.Limit, c.Cursor, c.All, c.Oldest, c.Latest, c.Inclusive)
	if err != nil {
		return err
	}

	return printMessages(appCtx, messages)
}

func (c *ConversationsRepliesCmd) Run(appCtx *app.Context) error {
	if c.Limit <= 0 {
		return errors.New("--limit must be greater than zero")
	}

	client, err := appCtx.SlackClient()
	if err != nil {
		return err
	}

	ctx, cancel := appCtx.RequestContext()
	defer cancel()

	channelID, err := appCtx.ResolveConversation(c.Channel)
	if err != nil {
		return err
	}

	messages, err := collectReplies(ctx, client, channelID, c.TS, c.Limit, c.Cursor, c.All, c.Oldest, c.Latest, c.Inclusive)
	if err != nil {
		return err
	}

	return printMessages(appCtx, messages)
}

func (c *ConversationsMembersCmd) Run(appCtx *app.Context) error {
	if c.Limit <= 0 {
		return errors.New("--limit must be greater than zero")
	}

	client, err := appCtx.SlackClient()
	if err != nil {
		return err
	}

	ctx, cancel := appCtx.RequestContext()
	defer cancel()

	channelID, err := appCtx.ResolveConversation(c.Channel)
	if err != nil {
		return err
	}

	members, err := collectMembers(ctx, client, channelID, c.Limit, c.Cursor, c.All)
	if err != nil {
		return err
	}

	switch appCtx.Output.Mode() {
	case output.JSON:
		return appCtx.Output.JSON(members)
	case output.Plain:
		appCtx.Output.Lines(members)
		return nil
	default:
		rows := make([][]string, 0, len(members))
		for _, member := range members {
			rows = append(rows, []string{member})
		}
		return appCtx.Output.Table([]string{"User ID"}, rows)
	}
}

func (c *ConversationsOpenCmd) Run(appCtx *app.Context) error {
	if len(c.Users) == 0 {
		return errors.New("at least one user reference is required")
	}

	client, err := appCtx.SlackClient()
	if err != nil {
		return err
	}

	ctx, cancel := appCtx.RequestContext()
	defer cancel()

	userIDs, err := appCtx.ResolveUsers(c.Users)
	if err != nil {
		return err
	}

	response, err := client.ConversationsOpen(ctx, userIDs, true)
	if err != nil {
		return err
	}

	if appCtx.Output.Mode() == output.JSON {
		return appCtx.Output.JSON(response)
	}

	if err := printConversation(appCtx, response.Channel); err != nil {
		return err
	}

	if appCtx.Output.Mode() == output.Plain {
		appCtx.Output.Lines([]string{
			"already_open=" + strconv.FormatBool(response.AlreadyOpen),
			"no_op=" + strconv.FormatBool(response.NoOp),
		})
		return nil
	}

	appCtx.Output.Lines([]string{
		"Already open: " + strconv.FormatBool(response.AlreadyOpen),
		"No-op: " + strconv.FormatBool(response.NoOp),
	})
	return nil
}

func (c *ChatDMCmd) Run(appCtx *app.Context) error {
	if len(c.Users) < 2 {
		return errors.New("usage: chat dm <user> [<user> ...] <text>")
	}

	text := strings.TrimSpace(c.Users[len(c.Users)-1])
	users := c.Users[:len(c.Users)-1]
	if text == "" {
		return errors.New("message text cannot be empty")
	}

	client, err := appCtx.SlackClient()
	if err != nil {
		return err
	}

	ctx, cancel := appCtx.RequestContext()
	defer cancel()

	userIDs, err := appCtx.ResolveUsers(users)
	if err != nil {
		return err
	}

	opened, err := client.ConversationsOpen(ctx, userIDs, true)
	if err != nil {
		return err
	}

	response, err := client.ChatPostMessage(ctx, opened.Channel.ID, text, "")
	if err != nil {
		return err
	}

	if appCtx.Output.Mode() == output.JSON {
		return appCtx.Output.JSON(map[string]any{
			"opened":  opened,
			"message": response,
		})
	}

	if err := printPostedMessage(appCtx, response.Channel, response.Ts, response.Message.Text); err != nil {
		return err
	}

	if appCtx.Output.Mode() == output.Plain {
		appCtx.Output.Lines([]string{
			"already_open=" + strconv.FormatBool(opened.AlreadyOpen),
			"no_op=" + strconv.FormatBool(opened.NoOp),
		})
		return nil
	}

	appCtx.Output.Lines([]string{
		"Opened: " + opened.Channel.ID,
		"Already open: " + strconv.FormatBool(opened.AlreadyOpen),
	})
	return nil
}

func (c *ConversationsJoinCmd) Run(appCtx *app.Context) error {
	client, err := appCtx.SlackClient()
	if err != nil {
		return err
	}

	ctx, cancel := appCtx.RequestContext()
	defer cancel()

	channelID, err := appCtx.ResolveConversation(c.Channel)
	if err != nil {
		return err
	}

	response, err := client.ConversationsJoin(ctx, channelID)
	if err != nil {
		return err
	}

	return printConversation(appCtx, response.Channel)
}

func (c *ConversationsLeaveCmd) Run(appCtx *app.Context) error {
	client, err := appCtx.SlackClient()
	if err != nil {
		return err
	}

	ctx, cancel := appCtx.RequestContext()
	defer cancel()

	channelID, err := appCtx.ResolveConversation(c.Channel)
	if err != nil {
		return err
	}

	response, err := client.ConversationsLeave(ctx, channelID)
	if err != nil {
		return err
	}

	switch appCtx.Output.Mode() {
	case output.JSON:
		return appCtx.Output.JSON(response)
	case output.Plain:
		appCtx.Output.Lines([]string{"not_in_channel=" + strconv.FormatBool(response.NotInChannel)})
	default:
		appCtx.Output.Lines([]string{"Left conversation", "Not in channel: " + strconv.FormatBool(response.NotInChannel)})
	}

	return nil
}

func (c *ConversationsCreateCmd) Run(appCtx *app.Context) error {
	client, err := appCtx.SlackClient()
	if err != nil {
		return err
	}

	ctx, cancel := appCtx.RequestContext()
	defer cancel()

	response, err := client.ConversationsCreate(ctx, c.Name, c.Private)
	if err != nil {
		return err
	}

	return printConversation(appCtx, response.Channel)
}

func (c *ConversationsRenameCmd) Run(appCtx *app.Context) error {
	client, err := appCtx.SlackClient()
	if err != nil {
		return err
	}

	ctx, cancel := appCtx.RequestContext()
	defer cancel()

	channelID, err := appCtx.ResolveConversation(c.Channel)
	if err != nil {
		return err
	}

	response, err := client.ConversationsRename(ctx, channelID, c.Name)
	if err != nil {
		return err
	}

	return printConversation(appCtx, response.Channel)
}

func (c *ConversationsTopicCmd) Run(appCtx *app.Context) error {
	client, err := appCtx.SlackClient()
	if err != nil {
		return err
	}

	ctx, cancel := appCtx.RequestContext()
	defer cancel()

	channelID, err := appCtx.ResolveConversation(c.Channel)
	if err != nil {
		return err
	}

	response, err := client.ConversationsSetTopic(ctx, channelID, c.Topic)
	if err != nil {
		return err
	}

	return printValueResponse(appCtx, "topic", response.Topic)
}

func (c *ConversationsPurposeCmd) Run(appCtx *app.Context) error {
	client, err := appCtx.SlackClient()
	if err != nil {
		return err
	}

	ctx, cancel := appCtx.RequestContext()
	defer cancel()

	channelID, err := appCtx.ResolveConversation(c.Channel)
	if err != nil {
		return err
	}

	response, err := client.ConversationsSetPurpose(ctx, channelID, c.Purpose)
	if err != nil {
		return err
	}

	return printValueResponse(appCtx, "purpose", response.Purpose)
}

func (c *ConversationsArchiveCmd) Run(appCtx *app.Context) error {
	channelID, err := appCtx.ResolveConversation(c.Channel)
	if err != nil {
		return err
	}

	return simpleConversationAction(appCtx, channelID, "Archived conversation", func(ctx context.Context, client *slackapi.Client) error {
		return client.ConversationsArchive(ctx, channelID)
	})
}

func (c *ConversationsUnarchiveCmd) Run(appCtx *app.Context) error {
	channelID, err := appCtx.ResolveConversation(c.Channel)
	if err != nil {
		return err
	}

	return simpleConversationAction(appCtx, channelID, "Unarchived conversation", func(ctx context.Context, client *slackapi.Client) error {
		return client.ConversationsUnarchive(ctx, channelID)
	})
}

func (c *ConversationsMarkCmd) Run(appCtx *app.Context) error {
	channelID, err := appCtx.ResolveConversation(c.Channel)
	if err != nil {
		return err
	}

	return simpleConversationAction(appCtx, channelID, "Updated read cursor", func(ctx context.Context, client *slackapi.Client) error {
		return client.ConversationsMark(ctx, channelID, c.TS)
	})
}

func (c *ConversationsUnreadCmd) Run(appCtx *app.Context) error {
	if c.Limit <= 0 {
		return errors.New("--limit must be greater than zero")
	}

	client, err := appCtx.SlackClient()
	if err != nil {
		return err
	}

	ctx, cancel := appCtx.RequestContext()
	defer cancel()

	channelID, err := appCtx.ResolveConversation(c.Channel)
	if err != nil {
		return err
	}

	info, err := client.ConversationsInfo(ctx, channelID, false)
	if err != nil {
		return err
	}
	if strings.TrimSpace(info.Channel.LastRead) == "" {
		return errors.New("Slack did not return last_read for this conversation/token, so unread messages cannot be derived")
	}

	messages, err := collectHistory(ctx, client, channelID, c.Limit, "", c.All, info.Channel.LastRead, "", false)
	if err != nil {
		return err
	}

	if appCtx.Output.Mode() == output.JSON {
		return appCtx.Output.JSON(map[string]any{
			"channel":                info.Channel,
			"derived_from_last_read": info.Channel.LastRead,
			"messages":               messages,
		})
	}

	if appCtx.Output.Mode() == output.Plain {
		appCtx.Output.Lines([]string{"last_read=" + info.Channel.LastRead})
		return printMessages(appCtx, messages)
	}

	appCtx.Output.Lines([]string{
		"Last read: " + info.Channel.LastRead,
		"Unread count display: " + strconv.Itoa(info.Channel.UnreadCountDisplay),
	})
	return printMessages(appCtx, messages)
}

func (c *ChatSendCmd) Run(appCtx *app.Context) error {
	client, err := appCtx.SlackClient()
	if err != nil {
		return err
	}

	ctx, cancel := appCtx.RequestContext()
	defer cancel()

	channelID, err := appCtx.ResolveConversation(c.Channel)
	if err != nil {
		return err
	}

	response, err := client.ChatPostMessage(ctx, channelID, c.Text, c.ThreadTS)
	if err != nil {
		return err
	}

	return printPostedMessage(appCtx, response.Channel, response.Ts, response.Message.Text)
}

func (c *ChatEphemeralCmd) Run(appCtx *app.Context) error {
	client, err := appCtx.SlackClient()
	if err != nil {
		return err
	}

	ctx, cancel := appCtx.RequestContext()
	defer cancel()

	channelID, err := appCtx.ResolveConversation(c.Channel)
	if err != nil {
		return err
	}
	userID, err := appCtx.ResolveUser(c.User)
	if err != nil {
		return err
	}

	response, err := client.ChatPostEphemeral(ctx, channelID, userID, c.Text, c.ThreadTS)
	if err != nil {
		return err
	}

	switch appCtx.Output.Mode() {
	case output.JSON:
		return appCtx.Output.JSON(response)
	case output.Plain:
		appCtx.Output.Lines([]string{"message_ts=" + response.MessageTs})
	default:
		appCtx.Output.Lines([]string{"Sent ephemeral message", "Timestamp: " + response.MessageTs})
	}

	return nil
}

func (c *ChatUpdateCmd) Run(appCtx *app.Context) error {
	client, err := appCtx.SlackClient()
	if err != nil {
		return err
	}

	ctx, cancel := appCtx.RequestContext()
	defer cancel()

	channelID, err := appCtx.ResolveConversation(c.Channel)
	if err != nil {
		return err
	}

	response, err := client.ChatUpdate(ctx, channelID, c.TS, c.Text)
	if err != nil {
		return err
	}

	return printPostedMessage(appCtx, response.Channel, response.Ts, response.Message.Text)
}

func (c *ChatDeleteCmd) Run(appCtx *app.Context) error {
	client, err := appCtx.SlackClient()
	if err != nil {
		return err
	}

	ctx, cancel := appCtx.RequestContext()
	defer cancel()

	channelID, err := appCtx.ResolveConversation(c.Channel)
	if err != nil {
		return err
	}

	response, err := client.ChatDelete(ctx, channelID, c.TS)
	if err != nil {
		return err
	}

	switch appCtx.Output.Mode() {
	case output.JSON:
		return appCtx.Output.JSON(response)
	case output.Plain:
		appCtx.Output.Lines([]string{"channel=" + response.Channel, "ts=" + response.Ts})
	default:
		appCtx.Output.Lines([]string{"Deleted message", "Channel: " + response.Channel, "Timestamp: " + response.Ts})
	}

	return nil
}

func (c *ChatPermalinkCmd) Run(appCtx *app.Context) error {
	client, err := appCtx.SlackClient()
	if err != nil {
		return err
	}

	ctx, cancel := appCtx.RequestContext()
	defer cancel()

	channelID, err := appCtx.ResolveConversation(c.Channel)
	if err != nil {
		return err
	}

	response, err := client.ChatGetPermalink(ctx, channelID, c.TS)
	if err != nil {
		return err
	}

	switch appCtx.Output.Mode() {
	case output.JSON:
		return appCtx.Output.JSON(response)
	case output.Plain:
		appCtx.Output.Lines([]string{"channel=" + response.Channel, "permalink=" + response.Permalink})
	default:
		appCtx.Output.Lines([]string{"Channel: " + response.Channel, "Permalink: " + response.Permalink})
	}

	return nil
}

func (c *SearchMessagesCmd) Run(appCtx *app.Context) error {
	if c.Count <= 0 {
		return errors.New("--count must be greater than zero")
	}
	if c.Page <= 0 {
		return errors.New("--page must be greater than zero")
	}

	client, err := appCtx.SlackClient()
	if err != nil {
		return err
	}

	ctx, cancel := appCtx.RequestContext()
	defer cancel()

	response, err := client.SearchMessages(ctx, c.Query, c.Count, c.Page, c.Sort, c.SortDir, c.Cursor, c.Highlight)
	if err != nil {
		return err
	}

	if appCtx.Output.Mode() == output.JSON {
		return appCtx.Output.JSON(response)
	}

	if appCtx.Output.Mode() == output.Plain {
		return appCtx.Output.Table(nil, searchRows(response.Messages.Matches))
	}

	appCtx.Output.Lines([]string{
		"Query: " + response.Query,
		"Total: " + strconv.Itoa(response.Messages.Total),
	})
	return appCtx.Output.Table([]string{"Channel", "User", "TS", "Text", "Permalink"}, searchRows(response.Messages.Matches))
}

func collectUsers(ctx context.Context, client *slackapi.Client, limit int, cursor string, all bool) ([]slackapi.User, error) {
	var users []slackapi.User
	for {
		response, err := client.UsersList(ctx, limit, cursor)
		if err != nil {
			return nil, err
		}
		users = append(users, response.Members...)
		if !all || response.ResponseMetadata.NextCursor == "" {
			break
		}
		cursor = response.ResponseMetadata.NextCursor
	}
	return users, nil
}

func collectConversations(ctx context.Context, client *slackapi.Client, limit int, cursor string, all bool, types string, excludeArchived bool) ([]slackapi.Conversation, error) {
	var conversations []slackapi.Conversation
	for {
		response, err := client.ConversationsList(ctx, limit, cursor, types, excludeArchived)
		if err != nil {
			return nil, err
		}
		conversations = append(conversations, response.Channels...)
		if !all || response.ResponseMetadata.NextCursor == "" {
			break
		}
		cursor = response.ResponseMetadata.NextCursor
	}
	return conversations, nil
}

func collectHistory(ctx context.Context, client *slackapi.Client, channel string, limit int, cursor string, all bool, oldest string, latest string, inclusive bool) ([]slackapi.Message, error) {
	var messages []slackapi.Message
	for {
		response, err := client.ConversationsHistory(ctx, channel, limit, cursor, oldest, latest, inclusive)
		if err != nil {
			return nil, err
		}
		messages = append(messages, response.Messages...)
		if !all || response.ResponseMetadata.NextCursor == "" {
			break
		}
		cursor = response.ResponseMetadata.NextCursor
	}
	return messages, nil
}

func collectReplies(ctx context.Context, client *slackapi.Client, channel string, ts string, limit int, cursor string, all bool, oldest string, latest string, inclusive bool) ([]slackapi.Message, error) {
	var messages []slackapi.Message
	for {
		response, err := client.ConversationsReplies(ctx, channel, ts, limit, cursor, oldest, latest, inclusive)
		if err != nil {
			return nil, err
		}
		messages = append(messages, response.Messages...)
		if !all || response.ResponseMetadata.NextCursor == "" {
			break
		}
		cursor = response.ResponseMetadata.NextCursor
	}
	return messages, nil
}

func collectMembers(ctx context.Context, client *slackapi.Client, channel string, limit int, cursor string, all bool) ([]string, error) {
	var members []string
	for {
		response, err := client.ConversationsMembers(ctx, channel, limit, cursor)
		if err != nil {
			return nil, err
		}
		members = append(members, response.Members...)
		if !all || response.ResponseMetadata.NextCursor == "" {
			break
		}
		cursor = response.ResponseMetadata.NextCursor
	}
	return members, nil
}

func parseParams(items []string) (url.Values, error) {
	params := url.Values{}
	for _, item := range items {
		key, value, ok := strings.Cut(item, "=")
		if !ok || strings.TrimSpace(key) == "" {
			return nil, fmt.Errorf("invalid --param %q, expected key=value", item)
		}
		params.Add(strings.TrimSpace(key), value)
	}
	return params, nil
}

func printUser(appCtx *app.Context, user slackapi.User) error {
	if appCtx.Output.Mode() == output.JSON {
		return appCtx.Output.JSON(user)
	}

	lines := []string{}
	if appCtx.Output.Mode() == output.Plain {
		lines = []string{
			"id=" + user.ID,
			"name=" + user.Name,
			"real_name=" + firstNonEmpty(user.RealName, user.Profile.RealName),
			"email=" + user.Profile.Email,
			"tz=" + user.TZ,
			"deleted=" + strconv.FormatBool(user.Deleted),
			"is_bot=" + strconv.FormatBool(user.IsBot),
		}
	} else {
		lines = []string{
			"ID: " + user.ID,
			"Name: " + user.Name,
			"Real name: " + firstNonEmpty(user.RealName, user.Profile.RealName),
			"Email: " + valueOr(user.Profile.Email, "n/a"),
			"TZ: " + valueOr(user.TZ, "n/a"),
			"Deleted: " + strconv.FormatBool(user.Deleted),
			"Bot: " + strconv.FormatBool(user.IsBot),
		}
	}
	appCtx.Output.Lines(lines)
	return nil
}

func printConversation(appCtx *app.Context, conversation slackapi.Conversation) error {
	if appCtx.Output.Mode() == output.JSON {
		return appCtx.Output.JSON(conversation)
	}

	lines := []string{}
	if appCtx.Output.Mode() == output.Plain {
		lines = []string{
			"id=" + conversation.ID,
			"name=" + firstNonEmpty(conversation.Name, conversation.NameNormalized),
			"type=" + conversationType(conversation),
			"is_private=" + strconv.FormatBool(conversation.IsPrivate),
			"is_archived=" + strconv.FormatBool(conversation.IsArchived),
			"num_members=" + formatInt(conversation.NumMembers),
			"topic=" + conversation.Topic.Value,
			"purpose=" + conversation.Purpose.Value,
			"last_read=" + conversation.LastRead,
			"unread_count=" + strconv.Itoa(conversation.UnreadCount),
			"unread_count_display=" + strconv.Itoa(conversation.UnreadCountDisplay),
		}
	} else {
		lines = []string{
			"ID: " + conversation.ID,
			"Name: " + firstNonEmpty(conversation.Name, conversation.NameNormalized),
			"Type: " + conversationType(conversation),
			"Private: " + strconv.FormatBool(conversation.IsPrivate),
			"Archived: " + strconv.FormatBool(conversation.IsArchived),
			"Members: " + valueOr(formatInt(conversation.NumMembers), "n/a"),
			"Topic: " + valueOr(conversation.Topic.Value, "n/a"),
			"Purpose: " + valueOr(conversation.Purpose.Value, "n/a"),
			"Last read: " + valueOr(conversation.LastRead, "n/a"),
			"Unread count: " + strconv.Itoa(conversation.UnreadCount),
			"Unread count display: " + strconv.Itoa(conversation.UnreadCountDisplay),
		}
	}
	appCtx.Output.Lines(lines)
	return nil
}

func printMessages(appCtx *app.Context, messages []slackapi.Message) error {
	switch appCtx.Output.Mode() {
	case output.JSON:
		return appCtx.Output.JSON(messages)
	case output.Plain:
		return appCtx.Output.Table(nil, messageRows(messages))
	default:
		return appCtx.Output.Table([]string{"TS", "User", "Subtype", "Replies", "Text"}, messageRows(messages))
	}
}

func printPostedMessage(appCtx *app.Context, channel string, ts string, text string) error {
	switch appCtx.Output.Mode() {
	case output.JSON:
		return appCtx.Output.JSON(map[string]any{"channel": channel, "ts": ts, "text": text})
	case output.Plain:
		appCtx.Output.Lines([]string{"channel=" + channel, "ts=" + ts, "text=" + text})
	default:
		appCtx.Output.Lines([]string{"Channel: " + channel, "Timestamp: " + ts, "Text: " + text})
	}
	return nil
}

func printValueResponse(appCtx *app.Context, key string, value string) error {
	if appCtx.Output.Mode() == output.JSON {
		return appCtx.Output.JSON(map[string]string{key: value})
	}
	if appCtx.Output.Mode() == output.Plain {
		appCtx.Output.Lines([]string{key + "=" + value})
		return nil
	}
	appCtx.Output.Lines([]string{strings.ToUpper(key[:1]) + key[1:] + ": " + value})
	return nil
}

func simpleConversationAction(appCtx *app.Context, channel string, label string, action func(context.Context, *slackapi.Client) error) error {
	client, err := appCtx.SlackClient()
	if err != nil {
		return err
	}

	ctx, cancel := appCtx.RequestContext()
	defer cancel()

	if err := action(ctx, client); err != nil {
		return err
	}

	if appCtx.Output.Mode() == output.JSON {
		return appCtx.Output.JSON(map[string]any{"channel": channel, "ok": true})
	}
	if appCtx.Output.Mode() == output.Plain {
		appCtx.Output.Lines([]string{"channel=" + channel, "ok=true"})
		return nil
	}
	appCtx.Output.Lines([]string{label, "Channel: " + channel})
	return nil
}

func userRows(users []slackapi.User) [][]string {
	rows := make([][]string, 0, len(users))
	for _, user := range users {
		rows = append(rows, []string{
			user.ID,
			user.Name,
			firstNonEmpty(user.RealName, user.Profile.RealName),
			user.Profile.Email,
			strconv.FormatBool(user.Deleted),
			strconv.FormatBool(user.IsBot),
		})
	}
	return rows
}

func conversationRows(conversations []slackapi.Conversation) [][]string {
	rows := make([][]string, 0, len(conversations))
	for _, conversation := range conversations {
		rows = append(rows, []string{
			conversation.ID,
			firstNonEmpty(conversation.Name, conversation.NameNormalized),
			conversationType(conversation),
			strconv.FormatBool(conversation.IsArchived),
			formatInt(conversation.NumMembers),
			formatInt(conversation.UnreadCountDisplay),
			truncate(conversation.Topic.Value, 60),
		})
	}
	return rows
}

func messageRows(messages []slackapi.Message) [][]string {
	rows := make([][]string, 0, len(messages))
	for _, message := range messages {
		rows = append(rows, []string{
			message.Ts,
			firstNonEmpty(message.User, message.BotID),
			message.Subtype,
			formatInt(message.ReplyCount),
			truncate(strings.ReplaceAll(message.Text, "\n", " "), 100),
		})
	}
	return rows
}

func searchRows(matches []slackapi.SearchMessageMatch) [][]string {
	rows := make([][]string, 0, len(matches))
	for _, match := range matches {
		rows = append(rows, []string{
			firstNonEmpty(match.Channel.Name, match.Channel.ID),
			firstNonEmpty(match.Username, match.User),
			match.Ts,
			truncate(strings.ReplaceAll(match.Text, "\n", " "), 80),
			match.Permalink,
		})
	}
	return rows
}

func conversationType(conversation slackapi.Conversation) string {
	switch {
	case conversation.IsIM:
		return "im"
	case conversation.IsMpIM:
		return "mpim"
	case conversation.IsPrivate:
		return "private"
	default:
		return "channel"
	}
}

func maskToken(token string) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return ""
	}
	if len(token) <= 10 {
		return token[:2] + strings.Repeat("*", len(token)-2)
	}
	return token[:6] + strings.Repeat("*", len(token)-10) + token[len(token)-4:]
}

func truncate(value string, limit int) string {
	if limit <= 0 || len(value) <= limit {
		return value
	}
	return value[:limit-3] + "..."
}

func formatInt(value int) string {
	if value == 0 {
		return ""
	}
	return strconv.Itoa(value)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func valueOr(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
