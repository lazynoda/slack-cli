package slack

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const defaultBaseURL = "https://slack.com/api"

type Options struct {
	Token               string
	Cookie              string
	BaseURL             string
	Timeout             time.Duration
	MaxRateLimitRetries int
}

type Client struct {
	token               string
	cookie              string
	baseURL             string
	httpClient          *http.Client
	maxRateLimitRetries int
}

type ResponseMetadata struct {
	NextCursor string   `json:"next_cursor,omitempty"`
	Warnings   []string `json:"warnings,omitempty"`
}

type apiEnvelope struct {
	OK               bool             `json:"ok"`
	Error            string           `json:"error,omitempty"`
	Warning          string           `json:"warning,omitempty"`
	Needed           string           `json:"needed,omitempty"`
	Provided         string           `json:"provided,omitempty"`
	ResponseMetadata ResponseMetadata `json:"response_metadata,omitempty"`
}

type APIError struct {
	Method   string
	Code     string
	Warning  string
	Needed   string
	Provided string
}

func (e *APIError) Error() string {
	parts := []string{fmt.Sprintf("slack %s failed: %s", e.Method, e.Code)}
	if e.Needed != "" {
		parts = append(parts, "needed="+e.Needed)
	}
	if e.Provided != "" {
		parts = append(parts, "provided="+e.Provided)
	}
	if e.Warning != "" {
		parts = append(parts, "warning="+e.Warning)
	}
	return strings.Join(parts, " ")
}

type AuthTestResponse struct {
	apiEnvelope
	URL    string `json:"url,omitempty"`
	Team   string `json:"team,omitempty"`
	User   string `json:"user,omitempty"`
	TeamID string `json:"team_id,omitempty"`
	UserID string `json:"user_id,omitempty"`
	BotID  string `json:"bot_id,omitempty"`
	AppID  string `json:"app_id,omitempty"`
	IsBot  bool   `json:"is_bot,omitempty"`
	Name   string `json:"name,omitempty"`
	Scope  string `json:"scope,omitempty"`
}

type UserProfile struct {
	DisplayName           string `json:"display_name,omitempty"`
	DisplayNameNormalized string `json:"display_name_normalized,omitempty"`
	RealName              string `json:"real_name,omitempty"`
	RealNameNormalized    string `json:"real_name_normalized,omitempty"`
	Email                 string `json:"email,omitempty"`
	Image24               string `json:"image_24,omitempty"`
	Image48               string `json:"image_48,omitempty"`
	Image72               string `json:"image_72,omitempty"`
}

type User struct {
	ID        string      `json:"id"`
	TeamID    string      `json:"team_id,omitempty"`
	Name      string      `json:"name"`
	RealName  string      `json:"real_name,omitempty"`
	TZ        string      `json:"tz,omitempty"`
	Deleted   bool        `json:"deleted,omitempty"`
	IsBot     bool        `json:"is_bot,omitempty"`
	IsAppUser bool        `json:"is_app_user,omitempty"`
	Profile   UserProfile `json:"profile,omitempty"`
}

type UsersListResponse struct {
	apiEnvelope
	Members []User `json:"members"`
}

type UsersInfoResponse struct {
	apiEnvelope
	User User `json:"user"`
}

type UsersPresenceResponse struct {
	apiEnvelope
	Presence        string `json:"presence,omitempty"`
	Online          bool   `json:"online,omitempty"`
	AutoAway        bool   `json:"auto_away,omitempty"`
	ManualAway      bool   `json:"manual_away,omitempty"`
	ConnectionCount int    `json:"connection_count,omitempty"`
	LastActivity    int64  `json:"last_activity,omitempty"`
}

type ConversationValue struct {
	Value string `json:"value"`
}

type Message struct {
	Type       string `json:"type,omitempty"`
	Subtype    string `json:"subtype,omitempty"`
	User       string `json:"user,omitempty"`
	BotID      string `json:"bot_id,omitempty"`
	Text       string `json:"text,omitempty"`
	Ts         string `json:"ts,omitempty"`
	ThreadTS   string `json:"thread_ts,omitempty"`
	ReplyCount int    `json:"reply_count,omitempty"`
}

type Conversation struct {
	ID                 string            `json:"id"`
	Name               string            `json:"name,omitempty"`
	NameNormalized     string            `json:"name_normalized,omitempty"`
	User               string            `json:"user,omitempty"`
	IsOpen             bool              `json:"is_open,omitempty"`
	IsIM               bool              `json:"is_im,omitempty"`
	IsMpIM             bool              `json:"is_mpim,omitempty"`
	IsGroup            bool              `json:"is_group,omitempty"`
	IsChannel          bool              `json:"is_channel,omitempty"`
	IsPrivate          bool              `json:"is_private,omitempty"`
	IsArchived         bool              `json:"is_archived,omitempty"`
	IsGeneral          bool              `json:"is_general,omitempty"`
	IsMember           bool              `json:"is_member,omitempty"`
	NumMembers         int               `json:"num_members,omitempty"`
	Topic              ConversationValue `json:"topic,omitempty"`
	Purpose            ConversationValue `json:"purpose,omitempty"`
	LastRead           string            `json:"last_read,omitempty"`
	UnreadCount        int               `json:"unread_count,omitempty"`
	UnreadCountDisplay int               `json:"unread_count_display,omitempty"`
	Latest             *Message          `json:"latest,omitempty"`
}

type ConversationsListResponse struct {
	apiEnvelope
	Channels []Conversation `json:"channels"`
}

type ConversationsInfoResponse struct {
	apiEnvelope
	Channel Conversation `json:"channel"`
}

type ConversationsHistoryResponse struct {
	apiEnvelope
	Messages []Message `json:"messages"`
	HasMore  bool      `json:"has_more,omitempty"`
}

type ConversationsRepliesResponse struct {
	apiEnvelope
	Messages []Message `json:"messages"`
	HasMore  bool      `json:"has_more,omitempty"`
}

type ConversationsMembersResponse struct {
	apiEnvelope
	Members []string `json:"members"`
}

type ConversationsOpenResponse struct {
	apiEnvelope
	Channel     Conversation `json:"channel"`
	AlreadyOpen bool         `json:"already_open,omitempty"`
	NoOp        bool         `json:"no_op,omitempty"`
}

type ConversationsCreateResponse struct {
	apiEnvelope
	Channel Conversation `json:"channel"`
}

type ConversationsJoinResponse struct {
	apiEnvelope
	Channel Conversation `json:"channel"`
}

type ConversationsLeaveResponse struct {
	apiEnvelope
	NotInChannel bool `json:"not_in_channel,omitempty"`
}

type ConversationsRenameResponse struct {
	apiEnvelope
	Channel Conversation `json:"channel"`
}

type ConversationsTopicResponse struct {
	apiEnvelope
	Topic string `json:"topic,omitempty"`
}

type ConversationsPurposeResponse struct {
	apiEnvelope
	Purpose string `json:"purpose,omitempty"`
}

type ChatPostMessageResponse struct {
	apiEnvelope
	Channel string  `json:"channel,omitempty"`
	Ts      string  `json:"ts,omitempty"`
	Message Message `json:"message,omitempty"`
}

type ChatPostEphemeralResponse struct {
	apiEnvelope
	MessageTs string `json:"message_ts,omitempty"`
}

type ChatDeleteResponse struct {
	apiEnvelope
	Channel string `json:"channel,omitempty"`
	Ts      string `json:"ts,omitempty"`
}

type ChatPermalinkResponse struct {
	apiEnvelope
	Channel   string `json:"channel,omitempty"`
	Permalink string `json:"permalink,omitempty"`
}

type SearchMessagesChannel struct {
	ID        string `json:"id,omitempty"`
	Name      string `json:"name,omitempty"`
	IsPrivate bool   `json:"is_private,omitempty"`
}

type SearchMessageMatch struct {
	Channel   SearchMessagesChannel `json:"channel,omitempty"`
	Permalink string                `json:"permalink,omitempty"`
	Text      string                `json:"text,omitempty"`
	Ts        string                `json:"ts,omitempty"`
	User      string                `json:"user,omitempty"`
	Username  string                `json:"username,omitempty"`
}

type SearchPaging struct {
	Count int `json:"count,omitempty"`
	Page  int `json:"page,omitempty"`
	Pages int `json:"pages,omitempty"`
	Total int `json:"total,omitempty"`
}

type SearchPagination struct {
	First      int `json:"first,omitempty"`
	Last       int `json:"last,omitempty"`
	Page       int `json:"page,omitempty"`
	PageCount  int `json:"page_count,omitempty"`
	PerPage    int `json:"per_page,omitempty"`
	TotalCount int `json:"total_count,omitempty"`
}

type SearchMessagesResults struct {
	Matches    []SearchMessageMatch `json:"matches"`
	Paging     SearchPaging         `json:"paging,omitempty"`
	Pagination SearchPagination     `json:"pagination,omitempty"`
	Total      int                  `json:"total,omitempty"`
	NextCursor string               `json:"next_cursor,omitempty"`
}

type SearchMessagesResponse struct {
	apiEnvelope
	Query    string                `json:"query,omitempty"`
	Messages SearchMessagesResults `json:"messages"`
}

func NewClient(opts Options) *Client {
	baseURL := strings.TrimRight(opts.BaseURL, "/")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	retries := opts.MaxRateLimitRetries
	if retries <= 0 {
		retries = 2
	}

	return &Client{
		token:   opts.Token,
		cookie:  strings.TrimSpace(opts.Cookie),
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		maxRateLimitRetries: retries,
	}
}

func (c *Client) Call(ctx context.Context, method string, params url.Values, out any) error {
	return c.CallWithHTTPMethod(ctx, method, http.MethodPost, params, out)
}

func (c *Client) CallWithHTTPMethod(ctx context.Context, method string, httpMethod string, params url.Values, out any) error {
	body, err := c.do(ctx, method, httpMethod, params)
	if err != nil {
		return err
	}

	var envelope apiEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return fmt.Errorf("decode slack %s envelope: %w", method, err)
	}
	if !envelope.OK {
		return &APIError{
			Method:   method,
			Code:     envelope.Error,
			Warning:  envelope.Warning,
			Needed:   envelope.Needed,
			Provided: envelope.Provided,
		}
	}

	if out == nil {
		return nil
	}

	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode slack %s response: %w", method, err)
	}

	return nil
}

func (c *Client) AuthTest(ctx context.Context) (*AuthTestResponse, error) {
	var response AuthTestResponse
	if err := c.Call(ctx, "auth.test", nil, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) UsersList(ctx context.Context, limit int, cursor string) (*UsersListResponse, error) {
	params := url.Values{}
	setInt(params, "limit", limit)
	setString(params, "cursor", cursor)

	var response UsersListResponse
	if err := c.CallWithHTTPMethod(ctx, "users.list", http.MethodGet, params, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) UsersInfo(ctx context.Context, user string) (*UsersInfoResponse, error) {
	params := url.Values{}
	params.Set("user", user)

	var response UsersInfoResponse
	if err := c.CallWithHTTPMethod(ctx, "users.info", http.MethodGet, params, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) UsersGetPresence(ctx context.Context, user string) (*UsersPresenceResponse, error) {
	params := url.Values{}
	params.Set("user", user)

	var response UsersPresenceResponse
	if err := c.CallWithHTTPMethod(ctx, "users.getPresence", http.MethodGet, params, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) ConversationsList(ctx context.Context, limit int, cursor string, types string, excludeArchived bool) (*ConversationsListResponse, error) {
	params := url.Values{}
	setInt(params, "limit", limit)
	setString(params, "cursor", cursor)
	setString(params, "types", types)
	setBool(params, "exclude_archived", excludeArchived)

	var response ConversationsListResponse
	if err := c.CallWithHTTPMethod(ctx, "conversations.list", http.MethodGet, params, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) ConversationsInfo(ctx context.Context, channel string, includeNumMembers bool) (*ConversationsInfoResponse, error) {
	params := url.Values{}
	params.Set("channel", channel)
	setBool(params, "include_num_members", includeNumMembers)

	var response ConversationsInfoResponse
	if err := c.CallWithHTTPMethod(ctx, "conversations.info", http.MethodGet, params, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) ConversationsHistory(ctx context.Context, channel string, limit int, cursor string, oldest string, latest string, inclusive bool) (*ConversationsHistoryResponse, error) {
	params := url.Values{}
	params.Set("channel", channel)
	setInt(params, "limit", limit)
	setString(params, "cursor", cursor)
	setString(params, "oldest", oldest)
	setString(params, "latest", latest)
	setBool(params, "inclusive", inclusive)

	var response ConversationsHistoryResponse
	if err := c.CallWithHTTPMethod(ctx, "conversations.history", http.MethodGet, params, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) ConversationsReplies(ctx context.Context, channel string, ts string, limit int, cursor string, oldest string, latest string, inclusive bool) (*ConversationsRepliesResponse, error) {
	params := url.Values{}
	params.Set("channel", channel)
	params.Set("ts", ts)
	setInt(params, "limit", limit)
	setString(params, "cursor", cursor)
	setString(params, "oldest", oldest)
	setString(params, "latest", latest)
	setBool(params, "inclusive", inclusive)

	var response ConversationsRepliesResponse
	if err := c.CallWithHTTPMethod(ctx, "conversations.replies", http.MethodGet, params, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) ConversationsMembers(ctx context.Context, channel string, limit int, cursor string) (*ConversationsMembersResponse, error) {
	params := url.Values{}
	params.Set("channel", channel)
	setInt(params, "limit", limit)
	setString(params, "cursor", cursor)

	var response ConversationsMembersResponse
	if err := c.CallWithHTTPMethod(ctx, "conversations.members", http.MethodGet, params, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) ConversationsOpen(ctx context.Context, users []string, returnIM bool) (*ConversationsOpenResponse, error) {
	params := url.Values{}
	params.Set("users", strings.Join(users, ","))
	setBool(params, "return_im", returnIM)

	var response ConversationsOpenResponse
	if err := c.Call(ctx, "conversations.open", params, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) ConversationsJoin(ctx context.Context, channel string) (*ConversationsJoinResponse, error) {
	params := url.Values{}
	params.Set("channel", channel)

	var response ConversationsJoinResponse
	if err := c.Call(ctx, "conversations.join", params, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) ConversationsLeave(ctx context.Context, channel string) (*ConversationsLeaveResponse, error) {
	params := url.Values{}
	params.Set("channel", channel)

	var response ConversationsLeaveResponse
	if err := c.Call(ctx, "conversations.leave", params, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) ConversationsCreate(ctx context.Context, name string, isPrivate bool) (*ConversationsCreateResponse, error) {
	params := url.Values{}
	params.Set("name", name)
	setBool(params, "is_private", isPrivate)

	var response ConversationsCreateResponse
	if err := c.Call(ctx, "conversations.create", params, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) ConversationsRename(ctx context.Context, channel string, name string) (*ConversationsRenameResponse, error) {
	params := url.Values{}
	params.Set("channel", channel)
	params.Set("name", name)

	var response ConversationsRenameResponse
	if err := c.Call(ctx, "conversations.rename", params, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) ConversationsSetTopic(ctx context.Context, channel string, topic string) (*ConversationsTopicResponse, error) {
	params := url.Values{}
	params.Set("channel", channel)
	params.Set("topic", topic)

	var response ConversationsTopicResponse
	if err := c.Call(ctx, "conversations.setTopic", params, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) ConversationsSetPurpose(ctx context.Context, channel string, purpose string) (*ConversationsPurposeResponse, error) {
	params := url.Values{}
	params.Set("channel", channel)
	params.Set("purpose", purpose)

	var response ConversationsPurposeResponse
	if err := c.Call(ctx, "conversations.setPurpose", params, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) ConversationsArchive(ctx context.Context, channel string) error {
	params := url.Values{}
	params.Set("channel", channel)
	return c.Call(ctx, "conversations.archive", params, &apiEnvelope{})
}

func (c *Client) ConversationsUnarchive(ctx context.Context, channel string) error {
	params := url.Values{}
	params.Set("channel", channel)
	return c.Call(ctx, "conversations.unarchive", params, &apiEnvelope{})
}

func (c *Client) ConversationsMark(ctx context.Context, channel string, ts string) error {
	params := url.Values{}
	params.Set("channel", channel)
	params.Set("ts", ts)
	return c.Call(ctx, "conversations.mark", params, &apiEnvelope{})
}

func (c *Client) ChatPostMessage(ctx context.Context, channel string, text string, threadTS string) (*ChatPostMessageResponse, error) {
	params := url.Values{}
	params.Set("channel", channel)
	params.Set("text", text)
	setString(params, "thread_ts", threadTS)

	var response ChatPostMessageResponse
	if err := c.Call(ctx, "chat.postMessage", params, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) ChatPostEphemeral(ctx context.Context, channel string, user string, text string, threadTS string) (*ChatPostEphemeralResponse, error) {
	params := url.Values{}
	params.Set("channel", channel)
	params.Set("user", user)
	params.Set("text", text)
	setString(params, "thread_ts", threadTS)

	var response ChatPostEphemeralResponse
	if err := c.Call(ctx, "chat.postEphemeral", params, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) ChatUpdate(ctx context.Context, channel string, ts string, text string) (*ChatPostMessageResponse, error) {
	params := url.Values{}
	params.Set("channel", channel)
	params.Set("ts", ts)
	params.Set("text", text)

	var response ChatPostMessageResponse
	if err := c.Call(ctx, "chat.update", params, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) ChatDelete(ctx context.Context, channel string, ts string) (*ChatDeleteResponse, error) {
	params := url.Values{}
	params.Set("channel", channel)
	params.Set("ts", ts)

	var response ChatDeleteResponse
	if err := c.Call(ctx, "chat.delete", params, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) ChatGetPermalink(ctx context.Context, channel string, messageTs string) (*ChatPermalinkResponse, error) {
	params := url.Values{}
	params.Set("channel", channel)
	params.Set("message_ts", messageTs)

	var response ChatPermalinkResponse
	if err := c.CallWithHTTPMethod(ctx, "chat.getPermalink", http.MethodGet, params, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) SearchMessages(ctx context.Context, query string, count int, page int, sort string, sortDir string, cursor string, highlight bool) (*SearchMessagesResponse, error) {
	params := url.Values{}
	params.Set("query", query)
	setInt(params, "count", count)
	if page > 1 {
		params.Set("page", strconv.Itoa(page))
	}
	setString(params, "sort", sort)
	setString(params, "sort_dir", sortDir)
	setString(params, "cursor", cursor)
	setBool(params, "highlight", highlight)

	var response SearchMessagesResponse
	if err := c.CallWithHTTPMethod(ctx, "search.messages", http.MethodGet, params, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) do(ctx context.Context, method string, httpMethod string, params url.Values) ([]byte, error) {
	encoded := ""
	if params != nil {
		encoded = params.Encode()
	}

	methodURL := c.baseURL + "/" + strings.TrimLeft(method, "/")
	httpMethod = strings.ToUpper(strings.TrimSpace(httpMethod))
	if httpMethod == "" {
		httpMethod = http.MethodPost
	}

	for attempt := 0; ; attempt++ {
		requestURL := methodURL
		var body io.Reader
		if httpMethod == http.MethodGet {
			if encoded != "" {
				requestURL += "?" + encoded
			}
		} else {
			body = strings.NewReader(encoded)
		}

		req, err := http.NewRequestWithContext(ctx, httpMethod, requestURL, body)
		if err != nil {
			return nil, fmt.Errorf("build slack %s request: %w", method, err)
		}
		req.Header.Set("Authorization", "Bearer "+c.token)
		if c.cookie != "" {
			req.Header.Set("Cookie", c.cookie)
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", "slack-cli/dev")
		if httpMethod != http.MethodGet {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("call slack %s: %w", method, err)
		}

		responseBody, readErr := io.ReadAll(resp.Body)
		closeErr := resp.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("read slack %s response: %w", method, readErr)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("close slack %s response: %w", method, closeErr)
		}

		if resp.StatusCode == http.StatusTooManyRequests && attempt < c.maxRateLimitRetries {
			retryAfter := time.Second
			if value := resp.Header.Get("Retry-After"); value != "" {
				if seconds, err := strconv.Atoi(value); err == nil && seconds > 0 {
					retryAfter = time.Duration(seconds) * time.Second
				}
			}

			timer := time.NewTimer(retryAfter)
			select {
			case <-ctx.Done():
				timer.Stop()
				return nil, ctx.Err()
			case <-timer.C:
				continue
			}
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("slack %s returned %s: %s", method, resp.Status, strings.TrimSpace(string(responseBody)))
		}

		return responseBody, nil
	}
}

func setString(params url.Values, key string, value string) {
	if strings.TrimSpace(value) != "" {
		params.Set(key, value)
	}
}

func setInt(params url.Values, key string, value int) {
	if value > 0 {
		params.Set(key, strconv.Itoa(value))
	}
}

func setBool(params url.Values, key string, value bool) {
	if value {
		params.Set(key, "true")
	}
}
