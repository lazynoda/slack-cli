package cli

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/steipete/sweetcookie"
	"github.com/syndtr/goleveldb/leveldb"

	"slack-cli/internal/app"
	"slack-cli/internal/output"
	slackapi "slack-cli/internal/slack"
)

func newTestAppContextNoToken(t *testing.T, mode output.Mode) *app.Context {
	t.Helper()
	t.Setenv("SLACK_CLI_TOKEN", "")
	t.Setenv("SLACK_TOKEN", "")
	ctx, err := app.NewContext(app.Settings{ConfigPath: filepath.Join(t.TempDir(), "config.json"), OutputMode: mode, Profile: "default"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("NewContext() error = %v", err)
	}
	ctx.Settings.Timeout = time.Second
	return ctx
}

func newResolvedTestContext(t *testing.T, mode output.Mode) (*app.Context, *bytes.Buffer) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth.test":
			_, _ = w.Write([]byte(`{"ok":true,"url":"https://example.test","team":"Team","team_id":"T1","user":"alice","user_id":"U1"}`))
		case "/users.list":
			_, _ = w.Write([]byte(`{"ok":true,"members":[{"id":"U1","name":"alice","real_name":"Alice Jones","profile":{"display_name":"aj","real_name":"Alice Jones"}}]}`))
		case "/users.info":
			_, _ = w.Write([]byte(`{"ok":true,"user":{"id":"U1","name":"alice","real_name":"Alice Jones","profile":{"display_name":"aj","real_name":"Alice Jones","email":"alice@example.com"}}}`))
		case "/users.getPresence":
			_, _ = w.Write([]byte(`{"ok":true,"presence":"active","online":true}`))
		case "/conversations.list":
			_, _ = w.Write([]byte(`{"ok":true,"channels":[{"id":"C1","name":"general","unread_count_display":2,"latest":{"ts":"2.0","text":"hello"}},{"id":"D1","is_im":true,"user":"U1","unread_count":1},{"id":"C2","name":"random","last_read":"1.0","latest":{"ts":"2.0","text":"random message"}}]} `))
		case "/conversations.info":
			_, _ = w.Write([]byte(`{"ok":true,"channel":{"id":"C1","name":"general","last_read":"1.0","unread_count_display":1}}`))
		case "/conversations.history":
			_, _ = w.Write([]byte(`{"ok":true,"messages":[{"ts":"2.0","text":"hello","user":"U1"}]}`))
		case "/conversations.replies":
			_, _ = w.Write([]byte(`{"ok":true,"messages":[{"ts":"2.1","text":"reply","user":"U1"}]}`))
		case "/conversations.members":
			_, _ = w.Write([]byte(`{"ok":true,"members":["U1"]}`))
		case "/conversations.open":
			_, _ = w.Write([]byte(`{"ok":true,"channel":{"id":"D1","is_im":true},"already_open":true}`))
		case "/conversations.join":
			_, _ = w.Write([]byte(`{"ok":true,"channel":{"id":"C1","name":"general"}}`))
		case "/conversations.leave":
			_, _ = w.Write([]byte(`{"ok":true,"not_in_channel":false}`))
		case "/conversations.create":
			_, _ = w.Write([]byte(`{"ok":true,"channel":{"id":"C2","name":"new-room"}}`))
		case "/conversations.rename":
			_, _ = w.Write([]byte(`{"ok":true,"channel":{"id":"C1","name":"renamed"}}`))
		case "/conversations.setTopic":
			_, _ = w.Write([]byte(`{"ok":true,"topic":"topic"}`))
		case "/conversations.setPurpose":
			_, _ = w.Write([]byte(`{"ok":true,"purpose":"purpose"}`))
		case "/conversations.archive", "/conversations.unarchive", "/conversations.mark":
			_, _ = w.Write([]byte(`{"ok":true}`))
		case "/chat.postMessage":
			_, _ = w.Write([]byte(`{"ok":true,"channel":"C1","ts":"1.2","message":{"text":"hi"}}`))
		case "/chat.postEphemeral":
			_, _ = w.Write([]byte(`{"ok":true,"message_ts":"1.3"}`))
		case "/chat.update":
			_, _ = w.Write([]byte(`{"ok":true,"channel":"C1","ts":"1.2","message":{"text":"updated"}}`))
		case "/chat.delete":
			_, _ = w.Write([]byte(`{"ok":true,"channel":"C1","ts":"1.2"}`))
		case "/chat.getPermalink":
			_, _ = w.Write([]byte(`{"ok":true,"channel":"C1","permalink":"https://example.test"}`))
		case "/search.messages":
			_, _ = w.Write([]byte(`{"ok":true,"query":"q","messages":{"matches":[{"channel":{"name":"general"},"username":"alice","ts":"1.0","text":"hit","permalink":"https://example.test"}],"total":1}}`))
		case "/reactions.add", "/reactions.remove":
			_, _ = w.Write([]byte(`{"ok":true}`))
		case "/reactions.get":
			_, _ = w.Write([]byte(`{"ok":true,"type":"message","channel":"C1","message":{"ts":"1.2","text":"hi","reactions":[{"name":"thumbsup","count":1,"users":["U1"]}]}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"ok":false,"error":"not_found"}`))
		}
	}))
	t.Cleanup(server.Close)
	t.Setenv("SLACK_CLI_TOKEN", "token")
	t.Setenv("SLACK_CLI_BASE_URL", server.URL)

	out := &bytes.Buffer{}
	ctx, err := app.NewContext(app.Settings{ConfigPath: filepath.Join(t.TempDir(), "config.json"), OutputMode: mode, Profile: "default", Timeout: time.Second}, out, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("NewContext() error = %v", err)
	}
	return ctx, out
}

func newServerTestContext(t *testing.T, mode output.Mode, handler http.HandlerFunc) *app.Context {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	t.Setenv("SLACK_CLI_TOKEN", "token")
	t.Setenv("SLACK_CLI_BASE_URL", server.URL)

	ctx, err := app.NewContext(app.Settings{ConfigPath: filepath.Join(t.TempDir(), "config.json"), OutputMode: mode, Profile: "default", Timeout: time.Second}, &bytes.Buffer{}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("NewContext() error = %v", err)
	}
	return ctx
}

func TestGlobalsSettings(t *testing.T) {
	settings, err := (&Globals{Profile: "p", Timeout: 2, JSON: true}).Settings()
	if err != nil {
		t.Fatalf("Settings() error = %v", err)
	}
	if settings.OutputMode != output.JSON {
		t.Fatalf("OutputMode = %v", settings.OutputMode)
	}

	if _, err := (&Globals{JSON: true, Plain: true}).Settings(); err == nil {
		t.Fatal("Settings() error = nil, want error")
	}
	if New() == nil {
		t.Fatal("New() = nil")
	}
}

func TestParseParamsAndHelpers(t *testing.T) {
	params, err := parseParams([]string{"a=1", "b=two"})
	if err != nil {
		t.Fatalf("parseParams() error = %v", err)
	}
	if params.Get("a") != "1" || params.Get("b") != "two" {
		t.Fatalf("params = %#v", params)
	}
	if _, err := parseParams([]string{"bad"}); err == nil {
		t.Fatal("parseParams() error = nil, want error")
	}
	if got := maskToken("xoxb-1234567890"); !strings.HasPrefix(got, "xoxb-1") {
		t.Fatalf("maskToken() = %q", got)
	}
	if truncate("abcdef", 4) != "a..." {
		t.Fatalf("truncate() unexpected")
	}
	if formatInt(0) != "" || formatInt(3) != "3" {
		t.Fatalf("formatInt() unexpected")
	}
	if firstNonEmpty("", "a", "b") != "a" || valueOr("", "x") != "x" {
		t.Fatalf("helper output unexpected")
	}
}

func TestRowsAndPrintHelpers(t *testing.T) {
	users := userRows([]slackapi.User{{ID: "U1", Name: "alice", Profile: slackapi.UserProfile{Email: "a@example.com"}}})
	conversations := conversationRows([]slackapi.Conversation{{ID: "C1", Name: "general", NumMembers: 3}})
	messages := messageRows([]slackapi.Message{{Ts: "1.0", User: "U1", Text: "hello\nworld", ReplyCount: 2}})
	search := searchRows([]slackapi.SearchMessageMatch{{Channel: slackapi.SearchMessagesChannel{Name: "general"}, Username: "alice", Ts: "1.0", Text: "hit", Permalink: "https://example.test"}})

	if len(users) != 1 || len(conversations) != 1 || len(messages) != 1 || len(search) != 1 {
		t.Fatal("row helpers returned unexpected lengths")
	}
	if conversationType(slackapi.Conversation{IsIM: true}) != "im" || conversationType(slackapi.Conversation{IsMpIM: true}) != "mpim" || conversationType(slackapi.Conversation{IsPrivate: true}) != "private" {
		t.Fatal("conversationType() unexpected")
	}

	var out bytes.Buffer
	ctx := &app.Context{Settings: app.Settings{OutputMode: output.Human}, Output: output.New(output.Human, &out, &bytes.Buffer{})}
	_ = printUser(ctx, slackapi.User{ID: "U1", Name: "alice"})
	_ = printConversation(ctx, slackapi.Conversation{ID: "C1", Name: "general"})
	_ = printMessages(ctx, []slackapi.Message{{Ts: "1.0", User: "U1", Text: "hello"}})
	_ = printPostedMessage(ctx, "C1", "1.2", "hi")
	_ = printValueResponse(ctx, "topic", "hello")
	if out.Len() == 0 {
		t.Fatal("print helpers wrote no output")
	}

	jsonCtx := &app.Context{Settings: app.Settings{OutputMode: output.JSON}, Output: output.New(output.JSON, &bytes.Buffer{}, &bytes.Buffer{})}
	if err := printUser(jsonCtx, slackapi.User{ID: "U1"}); err != nil {
		t.Fatalf("printUser(JSON) error = %v", err)
	}
	if err := printConversation(jsonCtx, slackapi.Conversation{ID: "C1"}); err != nil {
		t.Fatalf("printConversation(JSON) error = %v", err)
	}
	if err := printMessages(jsonCtx, []slackapi.Message{{Ts: "1.0"}}); err != nil {
		t.Fatalf("printMessages(JSON) error = %v", err)
	}
	if err := printPostedMessage(jsonCtx, "C1", "1.2", "hi"); err != nil {
		t.Fatalf("printPostedMessage(JSON) error = %v", err)
	}
	if err := printValueResponse(jsonCtx, "topic", "hello"); err != nil {
		t.Fatalf("printValueResponse(JSON) error = %v", err)
	}
	plainCtx := &app.Context{Settings: app.Settings{OutputMode: output.Plain}, Output: output.New(output.Plain, &bytes.Buffer{}, &bytes.Buffer{})}
	if err := printUser(plainCtx, slackapi.User{ID: "U1", Name: "alice"}); err != nil {
		t.Fatalf("printUser(Plain) error = %v", err)
	}
	if err := printConversation(plainCtx, slackapi.Conversation{ID: "C1", Name: "general"}); err != nil {
		t.Fatalf("printConversation(Plain) error = %v", err)
	}
	if err := printPostedMessage(plainCtx, "C1", "1.2", "hi"); err != nil {
		t.Fatalf("printPostedMessage(Plain) error = %v", err)
	}
	if err := printValueResponse(plainCtx, "topic", "hello"); err != nil {
		t.Fatalf("printValueResponse(Plain) error = %v", err)
	}
}

func TestValidationErrors(t *testing.T) {
	ctx := newTestAppContextNoToken(t, output.Human)
	if err := (&UsersListCmd{Limit: 0}).Run(ctx); err == nil {
		t.Fatal("UsersListCmd.Run() error = nil")
	}
	if err := (&ConversationsListCmd{Limit: 0}).Run(ctx); err == nil {
		t.Fatal("ConversationsListCmd.Run() error = nil")
	}
	if err := (&ConversationsHistoryCmd{Limit: 0}).Run(ctx); err == nil {
		t.Fatal("ConversationsHistoryCmd.Run() error = nil")
	}
	if err := (&ConversationsRepliesCmd{Limit: 0}).Run(ctx); err == nil {
		t.Fatal("ConversationsRepliesCmd.Run() error = nil")
	}
	if err := (&ConversationsMembersCmd{Limit: 0}).Run(ctx); err == nil {
		t.Fatal("ConversationsMembersCmd.Run() error = nil")
	}
	if err := (&ConversationsOpenCmd{}).Run(ctx); err == nil {
		t.Fatal("ConversationsOpenCmd.Run() error = nil")
	}
	if err := (&ChatDMCmd{Users: []string{"@alice"}}).Run(ctx); err == nil {
		t.Fatal("ChatDMCmd.Run() error = nil")
	}
	if err := (&ConversationsUnreadCmd{Limit: 0}).Run(ctx); err == nil {
		t.Fatal("ConversationsUnreadCmd.Run() error = nil")
	}
	if err := (&SearchMessagesCmd{Count: 0, Page: 1}).Run(ctx); err == nil {
		t.Fatal("SearchMessagesCmd.Run() error = nil")
	}
	if err := (&SearchMessagesCmd{Count: 1, Page: 0}).Run(ctx); err == nil {
		t.Fatal("SearchMessagesCmd.Run() page error = nil")
	}
	if err := (&ConversationsInboxCmd{Limit: 0}).Run(ctx); err == nil {
		t.Fatal("ConversationsInboxCmd.Run() error = nil")
	}
	if err := (&ReactionsAddCmd{Emoji: "   "}).Run(ctx); err == nil {
		t.Fatal("ReactionsAddCmd.Run() error = nil")
	}
	if err := (&ReactionsRemoveCmd{Emoji: "   "}).Run(ctx); err == nil {
		t.Fatal("ReactionsRemoveCmd.Run() error = nil")
	}
	if err := (&AuthSetTokenCmd{Token: "   "}).Run(ctx); err == nil {
		t.Fatal("AuthSetTokenCmd.Run() error = nil")
	}
	if err := (&AuthSetCookieCmd{Cookie: "   "}).Run(ctx); err == nil {
		t.Fatal("AuthSetCookieCmd.Run() error = nil")
	}
	if err := (&AuthImportCookieCmd{Browser: "invalid"}).Run(ctx); err == nil {
		t.Fatal("AuthImportCookieCmd.Run() error = nil")
	}
	if err := simpleConversationAction(ctx, "C1", "done", func(_ context.Context, _ *slackapi.Client) error { return nil }); err == nil {
		t.Fatal("simpleConversationAction() error = nil")
	}
}

func TestCommandErrorsWithoutToken(t *testing.T) {
	ctx := newTestAppContextNoToken(t, output.Human)

	checks := []struct {
		name string
		run  func() error
	}{
		{"auth test", func() error { return (&AuthTestCmd{}).Run(ctx) }},
		{"api call", func() error { return (&APICallCmd{Method: "auth.test", HTTPMethod: "POST"}).Run(ctx) }},
		{"resolve user", func() error { return (&ResolveUserCmd{User: "@alice"}).Run(ctx) }},
		{"resolve conversation", func() error { return (&ResolveConversationCmd{Channel: "#general"}).Run(ctx) }},
		{"users list", func() error { return (&UsersListCmd{Limit: 1}).Run(ctx) }},
		{"users info", func() error { return (&UsersInfoCmd{User: "@alice"}).Run(ctx) }},
		{"users presence", func() error { return (&UsersPresenceCmd{User: "@alice"}).Run(ctx) }},
		{"conversations list", func() error { return (&ConversationsListCmd{Limit: 1}).Run(ctx) }},
		{"conversations info", func() error { return (&ConversationsInfoCmd{Channel: "#general"}).Run(ctx) }},
		{"conversations history", func() error { return (&ConversationsHistoryCmd{Channel: "#general", Limit: 1}).Run(ctx) }},
		{"conversations replies", func() error { return (&ConversationsRepliesCmd{Channel: "#general", TS: "1.0", Limit: 1}).Run(ctx) }},
		{"conversations members", func() error { return (&ConversationsMembersCmd{Channel: "#general", Limit: 1}).Run(ctx) }},
		{"conversations open", func() error { return (&ConversationsOpenCmd{Users: []string{"@alice"}}).Run(ctx) }},
		{"chat dm", func() error { return (&ChatDMCmd{Users: []string{"@alice", "hi"}}).Run(ctx) }},
		{"conversations join", func() error { return (&ConversationsJoinCmd{Channel: "#general"}).Run(ctx) }},
		{"conversations leave", func() error { return (&ConversationsLeaveCmd{Channel: "#general"}).Run(ctx) }},
		{"conversations create", func() error { return (&ConversationsCreateCmd{Name: "new-room"}).Run(ctx) }},
		{"conversations rename", func() error { return (&ConversationsRenameCmd{Channel: "#general", Name: "renamed"}).Run(ctx) }},
		{"conversations topic", func() error { return (&ConversationsTopicCmd{Channel: "#general", Topic: "topic"}).Run(ctx) }},
		{"conversations purpose", func() error { return (&ConversationsPurposeCmd{Channel: "#general", Purpose: "purpose"}).Run(ctx) }},
		{"conversations archive", func() error { return (&ConversationsArchiveCmd{Channel: "#general"}).Run(ctx) }},
		{"conversations unarchive", func() error { return (&ConversationsUnarchiveCmd{Channel: "#general"}).Run(ctx) }},
		{"conversations mark", func() error { return (&ConversationsMarkCmd{Channel: "#general", TS: "1.0"}).Run(ctx) }},
		{"conversations unread", func() error { return (&ConversationsUnreadCmd{Channel: "#general", Limit: 1}).Run(ctx) }},
		{"chat send", func() error { return (&ChatSendCmd{Channel: "#general", Text: "hi"}).Run(ctx) }},
		{"chat ephemeral", func() error { return (&ChatEphemeralCmd{Channel: "#general", User: "@alice", Text: "hi"}).Run(ctx) }},
		{"chat update", func() error { return (&ChatUpdateCmd{Channel: "#general", TS: "1.0", Text: "hi"}).Run(ctx) }},
		{"chat delete", func() error { return (&ChatDeleteCmd{Channel: "#general", TS: "1.0"}).Run(ctx) }},
		{"chat permalink", func() error { return (&ChatPermalinkCmd{Channel: "#general", TS: "1.0"}).Run(ctx) }},
		{"search messages", func() error { return (&SearchMessagesCmd{Query: "q", Count: 1, Page: 1}).Run(ctx) }},
		{"conversations inbox", func() error { return (&ConversationsInboxCmd{Limit: 1}).Run(ctx) }},
		{"reactions add", func() error { return (&ReactionsAddCmd{Channel: "#general", TS: "1.0", Emoji: "thumbsup"}).Run(ctx) }},
		{"reactions remove", func() error { return (&ReactionsRemoveCmd{Channel: "#general", TS: "1.0", Emoji: "thumbsup"}).Run(ctx) }},
		{"reactions list", func() error { return (&ReactionsListCmd{Channel: "#general", TS: "1.0"}).Run(ctx) }},
	}

	for _, check := range checks {
		if err := check.run(); err == nil {
			t.Fatalf("%s error = nil", check.name)
		}
	}
}

func TestCommandEndpointErrors(t *testing.T) {
	resolverCtx := newServerTestContext(t, output.Human, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/users.list":
			_, _ = w.Write([]byte(`{"ok":true,"members":[{"id":"U1","name":"alice","real_name":"Alice Jones","profile":{"display_name":"aj","real_name":"Alice Jones"}}]}`))
		case "/conversations.list":
			_, _ = w.Write([]byte(`{"ok":true,"channels":[{"id":"C1","name":"general"}]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"ok":false,"error":"not_found"}`))
		}
	})

	directCtx := newServerTestContext(t, output.Human, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"ok":false,"error":"not_found"}`))
	})

	checks := []struct {
		name string
		run  func() error
	}{
		{"auth test", func() error { return (&AuthTestCmd{}).Run(directCtx) }},
		{"api call", func() error { return (&APICallCmd{Method: "auth.test", HTTPMethod: "POST"}).Run(directCtx) }},
		{"resolve user", func() error { return (&ResolveUserCmd{User: "@alice"}).Run(resolverCtx) }},
		{"resolve conversation", func() error { return (&ResolveConversationCmd{Channel: "#general"}).Run(resolverCtx) }},
		{"users list", func() error { return (&UsersListCmd{Limit: 1}).Run(directCtx) }},
		{"users info", func() error { return (&UsersInfoCmd{User: "@alice"}).Run(resolverCtx) }},
		{"users presence", func() error { return (&UsersPresenceCmd{User: "@alice"}).Run(resolverCtx) }},
		{"conversations list", func() error { return (&ConversationsListCmd{Limit: 1}).Run(directCtx) }},
		{"conversations info", func() error { return (&ConversationsInfoCmd{Channel: "#general"}).Run(resolverCtx) }},
		{"conversations history", func() error { return (&ConversationsHistoryCmd{Channel: "#general", Limit: 1}).Run(resolverCtx) }},
		{"conversations replies", func() error {
			return (&ConversationsRepliesCmd{Channel: "#general", TS: "1.0", Limit: 1}).Run(resolverCtx)
		}},
		{"conversations members", func() error { return (&ConversationsMembersCmd{Channel: "#general", Limit: 1}).Run(resolverCtx) }},
		{"conversations open", func() error { return (&ConversationsOpenCmd{Users: []string{"@alice"}}).Run(resolverCtx) }},
		{"chat dm", func() error { return (&ChatDMCmd{Users: []string{"@alice", "hi"}}).Run(resolverCtx) }},
		{"conversations join", func() error { return (&ConversationsJoinCmd{Channel: "#general"}).Run(resolverCtx) }},
		{"conversations leave", func() error { return (&ConversationsLeaveCmd{Channel: "#general"}).Run(resolverCtx) }},
		{"conversations create", func() error { return (&ConversationsCreateCmd{Name: "new-room"}).Run(directCtx) }},
		{"conversations rename", func() error { return (&ConversationsRenameCmd{Channel: "#general", Name: "renamed"}).Run(resolverCtx) }},
		{"conversations topic", func() error { return (&ConversationsTopicCmd{Channel: "#general", Topic: "topic"}).Run(resolverCtx) }},
		{"conversations purpose", func() error {
			return (&ConversationsPurposeCmd{Channel: "#general", Purpose: "purpose"}).Run(resolverCtx)
		}},
		{"conversations archive", func() error { return (&ConversationsArchiveCmd{Channel: "#general"}).Run(resolverCtx) }},
		{"conversations unarchive", func() error { return (&ConversationsUnarchiveCmd{Channel: "#general"}).Run(resolverCtx) }},
		{"conversations mark", func() error { return (&ConversationsMarkCmd{Channel: "#general", TS: "1.0"}).Run(resolverCtx) }},
		{"conversations unread", func() error { return (&ConversationsUnreadCmd{Channel: "#general", Limit: 1}).Run(resolverCtx) }},
		{"chat send", func() error { return (&ChatSendCmd{Channel: "#general", Text: "hi"}).Run(resolverCtx) }},
		{"chat ephemeral", func() error {
			return (&ChatEphemeralCmd{Channel: "#general", User: "@alice", Text: "hi"}).Run(resolverCtx)
		}},
		{"chat update", func() error { return (&ChatUpdateCmd{Channel: "#general", TS: "1.0", Text: "hi"}).Run(resolverCtx) }},
		{"chat delete", func() error { return (&ChatDeleteCmd{Channel: "#general", TS: "1.0"}).Run(resolverCtx) }},
		{"chat permalink", func() error { return (&ChatPermalinkCmd{Channel: "#general", TS: "1.0"}).Run(resolverCtx) }},
		{"search messages", func() error { return (&SearchMessagesCmd{Query: "q", Count: 1, Page: 1}).Run(directCtx) }},
		{"conversations inbox", func() error { return (&ConversationsInboxCmd{Limit: 1}).Run(directCtx) }},
		{"reactions add", func() error {
			return (&ReactionsAddCmd{Channel: "#general", TS: "1.0", Emoji: "thumbsup"}).Run(resolverCtx)
		}},
		{"reactions remove", func() error {
			return (&ReactionsRemoveCmd{Channel: "#general", TS: "1.0", Emoji: "thumbsup"}).Run(resolverCtx)
		}},
		{"reactions list", func() error { return (&ReactionsListCmd{Channel: "#general", TS: "1.0"}).Run(resolverCtx) }},
	}

	for _, check := range checks {
		if err := check.run(); err == nil {
			t.Fatalf("%s error = nil", check.name)
		}
	}
}

func TestCommandRunsAgainstFakeSlack(t *testing.T) {
	ctx, out := newResolvedTestContext(t, output.Human)

	checks := []struct {
		name string
		run  func() error
	}{
		{"auth status", func() error { return (&AuthStatusCmd{}).Run(ctx) }},
		{"auth clear", func() error { return (&AuthClearCmd{}).Run(ctx) }},
		{"auth set-cookie", func() error { return (&AuthSetCookieCmd{Cookie: "d=test-cookie"}).Run(ctx) }},
		{"auth clear-cookie", func() error { return (&AuthClearCookieCmd{}).Run(ctx) }},
		{"auth set-token", func() error { return (&AuthSetTokenCmd{Token: "xoxb-test"}).Run(ctx) }},
		{"auth test", func() error { return (&AuthTestCmd{}).Run(ctx) }},
		{"api call", func() error { return (&APICallCmd{Method: "auth.test", HTTPMethod: "POST"}).Run(ctx) }},
		{"resolve user", func() error { return (&ResolveUserCmd{User: "@alice"}).Run(ctx) }},
		{"resolve conversation", func() error { return (&ResolveConversationCmd{Channel: "#general"}).Run(ctx) }},
		{"users list", func() error { return (&UsersListCmd{Limit: 10}).Run(ctx) }},
		{"users info", func() error { return (&UsersInfoCmd{User: "@alice"}).Run(ctx) }},
		{"users presence", func() error { return (&UsersPresenceCmd{User: "@alice"}).Run(ctx) }},
		{"conversations list", func() error { return (&ConversationsListCmd{Limit: 10, Types: "public_channel,im"}).Run(ctx) }},
		{"conversations info", func() error { return (&ConversationsInfoCmd{Channel: "#general"}).Run(ctx) }},
		{"conversations history", func() error { return (&ConversationsHistoryCmd{Channel: "#general", Limit: 10}).Run(ctx) }},
		{"conversations replies", func() error { return (&ConversationsRepliesCmd{Channel: "#general", TS: "1.0", Limit: 10}).Run(ctx) }},
		{"conversations members", func() error { return (&ConversationsMembersCmd{Channel: "#general", Limit: 10}).Run(ctx) }},
		{"conversations open", func() error { return (&ConversationsOpenCmd{Users: []string{"@alice"}}).Run(ctx) }},
		{"chat dm", func() error { return (&ChatDMCmd{Users: []string{"@alice", "hello"}}).Run(ctx) }},
		{"conversations join", func() error { return (&ConversationsJoinCmd{Channel: "#general"}).Run(ctx) }},
		{"conversations leave", func() error { return (&ConversationsLeaveCmd{Channel: "#general"}).Run(ctx) }},
		{"conversations create", func() error { return (&ConversationsCreateCmd{Name: "new-room"}).Run(ctx) }},
		{"conversations rename", func() error { return (&ConversationsRenameCmd{Channel: "#general", Name: "renamed"}).Run(ctx) }},
		{"conversations topic", func() error { return (&ConversationsTopicCmd{Channel: "#general", Topic: "topic"}).Run(ctx) }},
		{"conversations purpose", func() error { return (&ConversationsPurposeCmd{Channel: "#general", Purpose: "purpose"}).Run(ctx) }},
		{"conversations archive", func() error { return (&ConversationsArchiveCmd{Channel: "#general"}).Run(ctx) }},
		{"conversations unarchive", func() error { return (&ConversationsUnarchiveCmd{Channel: "#general"}).Run(ctx) }},
		{"conversations mark", func() error { return (&ConversationsMarkCmd{Channel: "#general", TS: "1.0"}).Run(ctx) }},
		{"conversations unread", func() error { return (&ConversationsUnreadCmd{Channel: "#general", Limit: 10}).Run(ctx) }},
		{"chat send", func() error { return (&ChatSendCmd{Channel: "#general", Text: "hi"}).Run(ctx) }},
		{"chat ephemeral", func() error { return (&ChatEphemeralCmd{Channel: "#general", User: "@alice", Text: "hi"}).Run(ctx) }},
		{"chat update", func() error { return (&ChatUpdateCmd{Channel: "#general", TS: "1.2", Text: "updated"}).Run(ctx) }},
		{"chat delete", func() error { return (&ChatDeleteCmd{Channel: "#general", TS: "1.2"}).Run(ctx) }},
		{"chat permalink", func() error { return (&ChatPermalinkCmd{Channel: "#general", TS: "1.2"}).Run(ctx) }},
		{"search messages", func() error {
			return (&SearchMessagesCmd{Query: "q", Count: 20, Page: 1, Sort: "score", SortDir: "desc"}).Run(ctx)
		}},
		{"conversations inbox", func() error { return (&ConversationsInboxCmd{Limit: 10}).Run(ctx) }},
		{"reactions add", func() error { return (&ReactionsAddCmd{Channel: "#general", TS: "1.2", Emoji: "thumbsup"}).Run(ctx) }},
		{"reactions remove", func() error { return (&ReactionsRemoveCmd{Channel: "#general", TS: "1.2", Emoji: "thumbsup"}).Run(ctx) }},
		{"reactions list", func() error { return (&ReactionsListCmd{Channel: "#general", TS: "1.2"}).Run(ctx) }},
	}

	for _, check := range checks {
		if err := check.run(); err != nil {
			t.Fatalf("%s error = %v", check.name, err)
		}
	}

	if out.Len() == 0 {
		t.Fatal("command output buffer is empty")
	}
}

func TestAdditionalCommandModes(t *testing.T) {
	jsonCtx, _ := newResolvedTestContext(t, output.JSON)
	plainCtx, _ := newResolvedTestContext(t, output.Plain)

	checks := []func() error{
		func() error { return (&AuthStatusCmd{}).Run(jsonCtx) },
		func() error { return (&AuthStatusCmd{}).Run(plainCtx) },
		func() error { return (&AuthClearCmd{}).Run(jsonCtx) },
		func() error { return (&AuthClearCmd{}).Run(plainCtx) },
		func() error { return (&AuthSetCookieCmd{Cookie: "d=test-cookie"}).Run(jsonCtx) },
		func() error { return (&AuthSetCookieCmd{Cookie: "d=test-cookie"}).Run(plainCtx) },
		func() error { return (&AuthClearCookieCmd{}).Run(jsonCtx) },
		func() error { return (&AuthClearCookieCmd{}).Run(plainCtx) },
		func() error { return (&AuthSetTokenCmd{Token: "xoxb-test"}).Run(jsonCtx) },
		func() error { return (&AuthSetTokenCmd{Token: "xoxb-test"}).Run(plainCtx) },
		func() error { return (&AuthTestCmd{}).Run(jsonCtx) },
		func() error { return (&AuthTestCmd{}).Run(plainCtx) },
		func() error { return (&ResolveUserCmd{User: "@alice"}).Run(jsonCtx) },
		func() error { return (&ResolveUserCmd{User: "@alice"}).Run(plainCtx) },
		func() error { return (&ResolveConversationCmd{Channel: "#general"}).Run(jsonCtx) },
		func() error { return (&ResolveConversationCmd{Channel: "#general"}).Run(plainCtx) },
		func() error { return (&UsersListCmd{Limit: 10}).Run(jsonCtx) },
		func() error { return (&UsersListCmd{Limit: 10}).Run(plainCtx) },
		func() error { return (&UsersInfoCmd{User: "@alice"}).Run(jsonCtx) },
		func() error { return (&UsersInfoCmd{User: "@alice"}).Run(plainCtx) },
		func() error { return (&UsersPresenceCmd{User: "@alice"}).Run(jsonCtx) },
		func() error { return (&UsersPresenceCmd{User: "@alice"}).Run(plainCtx) },
		func() error { return (&ConversationsListCmd{Limit: 10, Types: "public_channel,im"}).Run(jsonCtx) },
		func() error { return (&ConversationsListCmd{Limit: 10, Types: "public_channel,im"}).Run(plainCtx) },
		func() error { return (&ConversationsInfoCmd{Channel: "#general"}).Run(jsonCtx) },
		func() error { return (&ConversationsInfoCmd{Channel: "#general"}).Run(plainCtx) },
		func() error { return (&ConversationsHistoryCmd{Channel: "#general", Limit: 10}).Run(jsonCtx) },
		func() error { return (&ConversationsHistoryCmd{Channel: "#general", Limit: 10}).Run(plainCtx) },
		func() error {
			return (&ConversationsRepliesCmd{Channel: "#general", TS: "1.0", Limit: 10}).Run(jsonCtx)
		},
		func() error {
			return (&ConversationsRepliesCmd{Channel: "#general", TS: "1.0", Limit: 10}).Run(plainCtx)
		},
		func() error { return (&ConversationsMembersCmd{Channel: "#general", Limit: 10}).Run(jsonCtx) },
		func() error { return (&ConversationsMembersCmd{Channel: "#general", Limit: 10}).Run(plainCtx) },
		func() error { return (&ConversationsOpenCmd{Users: []string{"@alice"}}).Run(jsonCtx) },
		func() error { return (&ConversationsOpenCmd{Users: []string{"@alice"}}).Run(plainCtx) },
		func() error { return (&ChatDMCmd{Users: []string{"@alice", "hello"}}).Run(jsonCtx) },
		func() error { return (&ChatDMCmd{Users: []string{"@alice", "hello"}}).Run(plainCtx) },
		func() error { return (&ConversationsJoinCmd{Channel: "#general"}).Run(jsonCtx) },
		func() error { return (&ConversationsJoinCmd{Channel: "#general"}).Run(plainCtx) },
		func() error { return (&ConversationsCreateCmd{Name: "new-room"}).Run(jsonCtx) },
		func() error { return (&ConversationsCreateCmd{Name: "new-room"}).Run(plainCtx) },
		func() error { return (&ConversationsRenameCmd{Channel: "#general", Name: "renamed"}).Run(jsonCtx) },
		func() error { return (&ConversationsRenameCmd{Channel: "#general", Name: "renamed"}).Run(plainCtx) },
		func() error { return (&ConversationsTopicCmd{Channel: "#general", Topic: "topic"}).Run(jsonCtx) },
		func() error { return (&ConversationsTopicCmd{Channel: "#general", Topic: "topic"}).Run(plainCtx) },
		func() error { return (&ConversationsPurposeCmd{Channel: "#general", Purpose: "purpose"}).Run(jsonCtx) },
		func() error { return (&ConversationsPurposeCmd{Channel: "#general", Purpose: "purpose"}).Run(plainCtx) },
		func() error { return (&ConversationsArchiveCmd{Channel: "#general"}).Run(jsonCtx) },
		func() error { return (&ConversationsArchiveCmd{Channel: "#general"}).Run(plainCtx) },
		func() error { return (&ConversationsUnarchiveCmd{Channel: "#general"}).Run(jsonCtx) },
		func() error { return (&ConversationsUnarchiveCmd{Channel: "#general"}).Run(plainCtx) },
		func() error { return (&ConversationsMarkCmd{Channel: "#general", TS: "1.0"}).Run(jsonCtx) },
		func() error { return (&ConversationsMarkCmd{Channel: "#general", TS: "1.0"}).Run(plainCtx) },
		func() error { return (&ConversationsLeaveCmd{Channel: "#general"}).Run(jsonCtx) },
		func() error { return (&ConversationsLeaveCmd{Channel: "#general"}).Run(plainCtx) },
		func() error { return (&ConversationsUnreadCmd{Channel: "#general", Limit: 10}).Run(jsonCtx) },
		func() error { return (&ConversationsUnreadCmd{Channel: "#general", Limit: 10}).Run(plainCtx) },
		func() error { return (&ChatSendCmd{Channel: "#general", Text: "hi"}).Run(jsonCtx) },
		func() error { return (&ChatSendCmd{Channel: "#general", Text: "hi"}).Run(plainCtx) },
		func() error { return (&ChatEphemeralCmd{Channel: "#general", User: "@alice", Text: "hi"}).Run(jsonCtx) },
		func() error {
			return (&ChatEphemeralCmd{Channel: "#general", User: "@alice", Text: "hi"}).Run(plainCtx)
		},
		func() error { return (&ChatUpdateCmd{Channel: "#general", TS: "1.2", Text: "updated"}).Run(jsonCtx) },
		func() error { return (&ChatUpdateCmd{Channel: "#general", TS: "1.2", Text: "updated"}).Run(plainCtx) },
		func() error { return (&ChatDeleteCmd{Channel: "#general", TS: "1.2"}).Run(jsonCtx) },
		func() error { return (&ChatDeleteCmd{Channel: "#general", TS: "1.2"}).Run(plainCtx) },
		func() error { return (&ChatPermalinkCmd{Channel: "#general", TS: "1.2"}).Run(jsonCtx) },
		func() error { return (&ChatPermalinkCmd{Channel: "#general", TS: "1.2"}).Run(plainCtx) },
		func() error {
			return (&SearchMessagesCmd{Query: "q", Count: 20, Page: 1, Sort: "score", SortDir: "desc"}).Run(jsonCtx)
		},
		func() error {
			return (&SearchMessagesCmd{Query: "q", Count: 20, Page: 1, Sort: "score", SortDir: "desc"}).Run(plainCtx)
		},
		func() error { return (&ConversationsInboxCmd{Limit: 10}).Run(jsonCtx) },
		func() error { return (&ConversationsInboxCmd{Limit: 10}).Run(plainCtx) },
		func() error {
			return (&ReactionsAddCmd{Channel: "#general", TS: "1.2", Emoji: "thumbsup"}).Run(jsonCtx)
		},
		func() error {
			return (&ReactionsAddCmd{Channel: "#general", TS: "1.2", Emoji: "thumbsup"}).Run(plainCtx)
		},
		func() error {
			return (&ReactionsRemoveCmd{Channel: "#general", TS: "1.2", Emoji: "thumbsup"}).Run(jsonCtx)
		},
		func() error {
			return (&ReactionsRemoveCmd{Channel: "#general", TS: "1.2", Emoji: "thumbsup"}).Run(plainCtx)
		},
		func() error { return (&ReactionsListCmd{Channel: "#general", TS: "1.2"}).Run(jsonCtx) },
		func() error { return (&ReactionsListCmd{Channel: "#general", TS: "1.2"}).Run(plainCtx) },
		func() error { return (&ReactionsListCmd{Channel: "#general", TS: "1.2", Full: true}).Run(jsonCtx) },
		func() error { return (&ReactionsListCmd{Channel: "#general", TS: "1.2", Full: true}).Run(plainCtx) },
	}

	for i, check := range checks {
		if err := check(); err != nil {
			t.Fatalf("additional mode check %d error = %v", i, err)
		}
	}
}

func TestCollectorHelpersAndUnreadError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/users.list":
			if r.URL.Query().Get("cursor") == "next" {
				_, _ = w.Write([]byte(`{"ok":true,"members":[{"id":"U2","name":"bob"}],"response_metadata":{"next_cursor":""}}`))
				return
			}
			_, _ = w.Write([]byte(`{"ok":true,"members":[{"id":"U1","name":"alice"}],"response_metadata":{"next_cursor":"next"}}`))
		case "/conversations.list":
			if r.URL.Query().Get("cursor") == "next" {
				_, _ = w.Write([]byte(`{"ok":true,"channels":[{"id":"C2","name":"random"}],"response_metadata":{"next_cursor":""}}`))
				return
			}
			_, _ = w.Write([]byte(`{"ok":true,"channels":[{"id":"C1","name":"general"}],"response_metadata":{"next_cursor":"next"}}`))
		case "/conversations.history":
			if r.URL.Query().Get("cursor") == "next" {
				_, _ = w.Write([]byte(`{"ok":true,"messages":[{"ts":"2.0","text":"later"}],"response_metadata":{"next_cursor":""}}`))
				return
			}
			_, _ = w.Write([]byte(`{"ok":true,"messages":[{"ts":"1.0","text":"first"}],"response_metadata":{"next_cursor":"next"}}`))
		case "/conversations.replies":
			if r.URL.Query().Get("cursor") == "next" {
				_, _ = w.Write([]byte(`{"ok":true,"messages":[{"ts":"1.2","text":"reply2"}],"response_metadata":{"next_cursor":""}}`))
				return
			}
			_, _ = w.Write([]byte(`{"ok":true,"messages":[{"ts":"1.1","text":"reply1"}],"response_metadata":{"next_cursor":"next"}}`))
		case "/conversations.members":
			if r.URL.Query().Get("cursor") == "next" {
				_, _ = w.Write([]byte(`{"ok":true,"members":["U2"],"response_metadata":{"next_cursor":""}}`))
				return
			}
			_, _ = w.Write([]byte(`{"ok":true,"members":["U1"],"response_metadata":{"next_cursor":"next"}}`))
		case "/conversations.info":
			_, _ = w.Write([]byte(`{"ok":true,"channel":{"id":"C1","name":"general"}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"ok":false,"error":"not_found"}`))
		}
	}))
	defer server.Close()

	client := slackapi.NewClient(slackapi.Options{Token: "token", BaseURL: server.URL, Timeout: time.Second})
	ctx := context.Background()

	if users, err := collectUsers(ctx, client, 10, "", true); err != nil || len(users) != 2 {
		t.Fatalf("collectUsers() = (%v, %v)", users, err)
	}
	if channels, err := collectConversations(ctx, client, 10, "", true, "public_channel", false); err != nil || len(channels) != 2 {
		t.Fatalf("collectConversations() = (%v, %v)", channels, err)
	}
	if messages, err := collectHistory(ctx, client, "C1", 10, "", true, "", "", false); err != nil || len(messages) != 2 {
		t.Fatalf("collectHistory() = (%v, %v)", messages, err)
	}
	if replies, err := collectReplies(ctx, client, "C1", "1.0", 10, "", true, "", "", false); err != nil || len(replies) != 2 {
		t.Fatalf("collectReplies() = (%v, %v)", replies, err)
	}
	if members, err := collectMembers(ctx, client, "C1", 10, "", true); err != nil || len(members) != 2 {
		t.Fatalf("collectMembers() = (%v, %v)", members, err)
	}

	t.Setenv("SLACK_CLI_TOKEN", "token")
	t.Setenv("SLACK_CLI_BASE_URL", server.URL)
	appCtx, err := app.NewContext(app.Settings{ConfigPath: filepath.Join(t.TempDir(), "config.json"), OutputMode: output.Human, Profile: "default", Timeout: time.Second}, &bytes.Buffer{}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("NewContext() error = %v", err)
	}
	if err := (&ConversationsUnreadCmd{Channel: "#general", Limit: 10}).Run(appCtx); err == nil {
		t.Fatal("ConversationsUnreadCmd.Run(last_read missing) error = nil")
	}
}

func TestAuthImportCookieCmd(t *testing.T) {
	origGet := sweetcookieGet
	defer func() { sweetcookieGet = origGet }()

	t.Run("success with token extraction", func(t *testing.T) {
		sweetcookieGet = func(ctx context.Context, opts sweetcookie.Options) (sweetcookie.Result, error) {
			return sweetcookie.Result{
				Cookies: []sweetcookie.Cookie{
					{
						Name:  "d",
						Value: "cookie-secret-value",
					},
				},
			}, nil
		}

		tempHome := t.TempDir()
		t.Setenv("HOME", tempHome)
		t.Setenv("USERPROFILE", tempHome)

		// Set up mock LevelDB files for "Default" profile
		var chromeBase string
		switch runtime.GOOS {
		case "darwin":
			chromeBase = filepath.Join(tempHome, "Library/Application Support/Google/Chrome")
		case "linux":
			chromeBase = filepath.Join(tempHome, ".config/google-chrome")
		case "windows":
			chromeBase = filepath.Join(tempHome, "Google/Chrome/User Data")
		}

		if chromeBase != "" {
			leveldbPath := filepath.Join(chromeBase, "Default", "Local Storage", "leveldb")
			if err := os.MkdirAll(leveldbPath, 0755); err != nil {
				t.Fatalf("MkdirAll() error = %v", err)
			}
			db, err := leveldb.OpenFile(leveldbPath, nil)
			if err != nil {
				t.Fatalf("leveldb.OpenFile() error = %v", err)
			}
			mockConfig := `{"teams":{"T1":{"token":"xoxc-mock-token-success"}}}`
			val := append([]byte{1}, []byte(mockConfig)...)
			if err := db.Put([]byte("_https://app.slack.com\x00\x01localConfig_v2"), val, nil); err != nil {
				t.Fatalf("db.Put() error = %v", err)
			}
			if err := db.Close(); err != nil {
				t.Fatalf("db.Close() error = %v", err)
			}
		}

		appCtx := newTestAppContextNoToken(t, output.Human)
		cmd := &AuthImportCookieCmd{Browser: "chrome"}
		if err := cmd.Run(appCtx); err != nil {
			t.Fatalf("AuthImportCookieCmd.Run() error = %v", err)
		}

		if got := appCtx.Config.Profile("default").Cookie; got != "d=cookie-secret-value" {
			t.Fatalf("saved cookie = %q, want %q", got, "d=cookie-secret-value")
		}
		if chromeBase != "" {
			if gotToken := appCtx.Config.Profile("default").Token; gotToken != "xoxc-mock-token-success" {
				t.Fatalf("saved token = %q, want %q", gotToken, "xoxc-mock-token-success")
			}
		}
	})

	t.Run("success with explicit browser profile", func(t *testing.T) {
		sweetcookieGet = func(ctx context.Context, opts sweetcookie.Options) (sweetcookie.Result, error) {
			if opts.Profiles[sweetcookie.BrowserChrome] != "Profile 5" {
				return sweetcookie.Result{}, errors.New("expected Profile 5")
			}
			return sweetcookie.Result{
				Cookies: []sweetcookie.Cookie{
					{
						Name:  "d",
						Value: "cookie-secret-value-5",
					},
				},
			}, nil
		}

		tempHome := t.TempDir()
		t.Setenv("HOME", tempHome)
		t.Setenv("USERPROFILE", tempHome)

		var chromeBase string
		switch runtime.GOOS {
		case "darwin":
			chromeBase = filepath.Join(tempHome, "Library/Application Support/Google/Chrome")
		case "linux":
			chromeBase = filepath.Join(tempHome, ".config/google-chrome")
		case "windows":
			chromeBase = filepath.Join(tempHome, "Google/Chrome/User Data")
		}

		if chromeBase != "" {
			leveldbPath := filepath.Join(chromeBase, "Profile 5", "Local Storage", "leveldb")
			if err := os.MkdirAll(leveldbPath, 0755); err != nil {
				t.Fatalf("MkdirAll() error = %v", err)
			}
			db, err := leveldb.OpenFile(leveldbPath, nil)
			if err != nil {
				t.Fatalf("leveldb.OpenFile() error = %v", err)
			}
			mockConfig := `{"teams":{"T2":{"token":"xoxc-mock-token-5"}}}`
			val := append([]byte{1}, []byte(mockConfig)...)
			if err := db.Put([]byte("_https://app.slack.com\x00\x01localConfig_v2"), val, nil); err != nil {
				t.Fatalf("db.Put() error = %v", err)
			}
			if err := db.Close(); err != nil {
				t.Fatalf("db.Close() error = %v", err)
			}
		}

		appCtx := newTestAppContextNoToken(t, output.Human)
		cmd := &AuthImportCookieCmd{Browser: "chrome", BrowserProfile: "Profile 5"}
		if err := cmd.Run(appCtx); err != nil {
			t.Fatalf("AuthImportCookieCmd.Run() error = %v", err)
		}

		if got := appCtx.Config.Profile("default").Cookie; got != "d=cookie-secret-value-5" {
			t.Fatalf("saved cookie = %q, want %q", got, "d=cookie-secret-value-5")
		}
		if chromeBase != "" {
			if gotToken := appCtx.Config.Profile("default").Token; gotToken != "xoxc-mock-token-5" {
				t.Fatalf("saved token = %q, want %q", gotToken, "xoxc-mock-token-5")
			}
		}
	})

	t.Run("cookie not found", func(t *testing.T) {
		sweetcookieGet = func(ctx context.Context, opts sweetcookie.Options) (sweetcookie.Result, error) {
			return sweetcookie.Result{
				Cookies: []sweetcookie.Cookie{
					{
						Name:  "other",
						Value: "other-val",
					},
				},
			}, nil
		}

		appCtx := newTestAppContextNoToken(t, output.Human)
		cmd := &AuthImportCookieCmd{Browser: "chrome"}
		if err := cmd.Run(appCtx); err == nil {
			t.Fatal("AuthImportCookieCmd.Run() expected error when cookie is not found")
		}
	})

	t.Run("extraction error", func(t *testing.T) {
		sweetcookieGet = func(ctx context.Context, opts sweetcookie.Options) (sweetcookie.Result, error) {
			return sweetcookie.Result{}, errors.New("extract failed")
		}

		appCtx := newTestAppContextNoToken(t, output.Human)
		cmd := &AuthImportCookieCmd{Browser: "chrome"}
		if err := cmd.Run(appCtx); err == nil {
			t.Fatal("AuthImportCookieCmd.Run() expected error on sweetcookie.Get failure")
		}
	})
}

func TestGetBrowserUserDataDir(t *testing.T) {
	browsers := []string{"chrome", "brave", "edge", "chromium"}
	if runtime.GOOS == "darwin" {
		browsers = append(browsers, "vivaldi", "opera", "arc")
	}
	for _, b := range browsers {
		dir, err := getBrowserUserDataDir(b)
		if err != nil {
			t.Fatalf("getBrowserUserDataDir(%s) unexpected error: %v", b, err)
		}
		if dir == "" {
			t.Fatalf("getBrowserUserDataDir(%s) returned empty path", b)
		}
	}
	if _, err := getBrowserUserDataDir("invalid"); err == nil {
		t.Fatal("expected error for invalid browser")
	}
}

func TestExtractChromeToken(t *testing.T) {
	t.Run("missing leveldb", func(t *testing.T) {
		_, err := extractChromeToken(t.TempDir())
		if err == nil {
			t.Fatal("expected error when leveldb dir is missing")
		}
	})

	t.Run("empty leveldb", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, "Local Storage", "leveldb"), 0755); err != nil {
			t.Fatal(err)
		}
		_, err := extractChromeToken(dir)
		if err == nil {
			t.Fatal("expected error when leveldb is empty/missing files")
		}
	})

	t.Run("success with token", func(t *testing.T) {
		dir := t.TempDir()
		leveldbPath := filepath.Join(dir, "Local Storage", "leveldb")
		if err := os.MkdirAll(leveldbPath, 0755); err != nil {
			t.Fatalf("MkdirAll() error = %v", err)
		}

		db, err := leveldb.OpenFile(leveldbPath, nil)
		if err != nil {
			t.Fatalf("leveldb.OpenFile() error = %v", err)
		}

		mockConfig := `{"teams":{"T1":{"token":"xoxc-mock-token"}}}`
		val := append([]byte{1}, []byte(mockConfig)...)
		if err := db.Put([]byte("_https://app.slack.com\x00\x01localConfig_v2"), val, nil); err != nil {
			t.Fatalf("db.Put() error = %v", err)
		}
		_ = db.Close()

		token, err := extractChromeToken(dir)
		if err != nil {
			t.Fatalf("extractChromeToken() error = %v", err)
		}
		if token != "xoxc-mock-token" {
			t.Fatalf("extracted token = %q, want %q", token, "xoxc-mock-token")
		}
	})

	t.Run("no localConfig_v2 key", func(t *testing.T) {
		dir := t.TempDir()
		leveldbPath := filepath.Join(dir, "Local Storage", "leveldb")
		if err := os.MkdirAll(leveldbPath, 0755); err != nil {
			t.Fatalf("MkdirAll() error = %v", err)
		}

		db, err := leveldb.OpenFile(leveldbPath, nil)
		if err != nil {
			t.Fatalf("leveldb.OpenFile() error = %v", err)
		}
		if err := db.Put([]byte("someOtherKey"), []byte("someValue"), nil); err != nil {
			t.Fatalf("db.Put() error = %v", err)
		}
		_ = db.Close()

		_, err = extractChromeToken(dir)
		if err == nil || !strings.Contains(err.Error(), "localConfig_v2 not found") {
			t.Fatalf("expected 'localConfig_v2 not found' error, got: %v", err)
		}
	})

	t.Run("localConfig_v2 malformed json start", func(t *testing.T) {
		dir := t.TempDir()
		leveldbPath := filepath.Join(dir, "Local Storage", "leveldb")
		if err := os.MkdirAll(leveldbPath, 0755); err != nil {
			t.Fatalf("MkdirAll() error = %v", err)
		}

		db, err := leveldb.OpenFile(leveldbPath, nil)
		if err != nil {
			t.Fatalf("leveldb.OpenFile() error = %v", err)
		}
		if err := db.Put([]byte("_https://app.slack.com\x00\x01localConfig_v2"), []byte{1, 2, 3}, nil); err != nil {
			t.Fatalf("db.Put() error = %v", err)
		}
		_ = db.Close()

		_, err = extractChromeToken(dir)
		if err == nil || !strings.Contains(err.Error(), "localConfig_v2 not found") {
			t.Fatalf("expected error, got: %v", err)
		}
	})

	t.Run("localConfig_v2 invalid json format", func(t *testing.T) {
		dir := t.TempDir()
		leveldbPath := filepath.Join(dir, "Local Storage", "leveldb")
		if err := os.MkdirAll(leveldbPath, 0755); err != nil {
			t.Fatalf("MkdirAll() error = %v", err)
		}

		db, err := leveldb.OpenFile(leveldbPath, nil)
		if err != nil {
			t.Fatalf("leveldb.OpenFile() error = %v", err)
		}
		val := append([]byte{1}, []byte("{invalid json")...)
		if err := db.Put([]byte("_https://app.slack.com\x00\x01localConfig_v2"), val, nil); err != nil {
			t.Fatalf("db.Put() error = %v", err)
		}
		_ = db.Close()

		_, err = extractChromeToken(dir)
		if err == nil || !strings.Contains(err.Error(), "parse localConfig_v2 JSON") {
			t.Fatalf("expected parse error, got: %v", err)
		}
	})

	t.Run("localConfig_v2 empty teams", func(t *testing.T) {
		dir := t.TempDir()
		leveldbPath := filepath.Join(dir, "Local Storage", "leveldb")
		if err := os.MkdirAll(leveldbPath, 0755); err != nil {
			t.Fatalf("MkdirAll() error = %v", err)
		}

		db, err := leveldb.OpenFile(leveldbPath, nil)
		if err != nil {
			t.Fatalf("leveldb.OpenFile() error = %v", err)
		}
		val := append([]byte{1}, []byte(`{"teams":{}}`)...)
		if err := db.Put([]byte("_https://app.slack.com\x00\x01localConfig_v2"), val, nil); err != nil {
			t.Fatalf("db.Put() error = %v", err)
		}
		_ = db.Close()

		_, err = extractChromeToken(dir)
		if err == nil || !strings.Contains(err.Error(), "no team tokens found") {
			t.Fatalf("expected 'no team tokens found' error, got: %v", err)
		}
	})
}

func TestIsNewer(t *testing.T) {
	if !isNewer("2.0", "1.0") {
		t.Error("expected 2.0 > 1.0")
	}
	if isNewer("1.0", "2.0") {
		t.Error("expected 1.0 < 2.0")
	}
	if !isNewer("abc", "abb") {
		t.Error("expected abc > abb (string fallback)")
	}
	if isNewer("1.0", "abc") {
		t.Error("expected fallback for mixed")
	}
}
