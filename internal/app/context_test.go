package app

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"slack-cli/internal/config"
	"slack-cli/internal/output"
	slackapi "slack-cli/internal/slack"
)

func newTestContext(t *testing.T) *Context {
	t.Helper()
	ctx, err := NewContext(Settings{ConfigPath: filepath.Join(t.TempDir(), "config.json"), OutputMode: output.Human, Profile: "default", Timeout: time.Second}, &bytes.Buffer{}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("NewContext() error = %v", err)
	}
	return ctx
}

func newSlackServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/users.list":
			_, _ = w.Write([]byte(`{"ok":true,"members":[{"id":"U1","name":"alice","real_name":"Alice Jones","profile":{"display_name":"aj","real_name":"Alice Jones"}},{"id":"U2","name":"bob","real_name":"Bob Smith","profile":{"display_name":"bobby","real_name":"Bob Smith"}}]}`))
		case "/conversations.list":
			_, _ = w.Write([]byte(`{"ok":true,"channels":[{"id":"C1","name":"general"},{"id":"G1","name":"random","is_private":true}]}`))
		case "/auth.test":
			_, _ = w.Write([]byte(`{"ok":true,"team":"Team"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(fmt.Sprintf(`{"ok":false,"error":"unknown_path:%s"}`, r.URL.Path)))
		}
	}))
}

func TestTokenPreferenceEnvThenConfig(t *testing.T) {
	t.Setenv("SLACK_CLI_TOKEN", "env-token")
	t.Setenv("SLACK_TOKEN", "fallback-token")

	ctx := newTestContext(t)
	ctx.Config.SetToken("default", "config-token")

	token, source := ctx.Token()
	if token != "env-token" || source != "SLACK_CLI_TOKEN" {
		t.Fatalf("Token() = (%q, %q)", token, source)
	}
}

func TestCookieAndBaseURLFromEnv(t *testing.T) {
	t.Setenv("SLACK_COOKIE", "d=abc")
	t.Setenv("SLACK_BASE_URL", "http://example.test")

	ctx := newTestContext(t)

	if cookie, source := ctx.Cookie(); cookie != "d=abc" || source != "SLACK_COOKIE" {
		t.Fatalf("Cookie() = (%q, %q)", cookie, source)
	}
	if baseURL, source := ctx.BaseURL(); baseURL != "http://example.test" || source != "SLACK_BASE_URL" {
		t.Fatalf("BaseURL() = (%q, %q)", baseURL, source)
	}
}

func TestTokenCookieAndBaseURLPrecedence(t *testing.T) {
	t.Setenv("SLACK_CLI_TOKEN", "")
	t.Setenv("SLACK_TOKEN", "env-fallback")
	t.Setenv("SLACK_CLI_COOKIE", "cli-cookie")
	t.Setenv("SLACK_COOKIE", "env-cookie")
	t.Setenv("SLACK_CLI_BASE_URL", "http://cli.test")
	t.Setenv("SLACK_BASE_URL", "http://env.test")

	ctx := newTestContext(t)
	ctx.Config.SetToken("default", "config-token")

	if token, source := ctx.Token(); token != "env-fallback" || source != "SLACK_TOKEN" {
		t.Fatalf("Token() fallback = (%q, %q)", token, source)
	}
	if cookie, source := ctx.Cookie(); cookie != "cli-cookie" || source != "SLACK_CLI_COOKIE" {
		t.Fatalf("Cookie() precedence = (%q, %q)", cookie, source)
	}
	if baseURL, source := ctx.BaseURL(); baseURL != "http://cli.test" || source != "SLACK_CLI_BASE_URL" {
		t.Fatalf("BaseURL() precedence = (%q, %q)", baseURL, source)
	}

	t.Setenv("SLACK_TOKEN", "")
	if token, source := ctx.Token(); token != "config-token" || source != "config" {
		t.Fatalf("Token() config fallback = (%q, %q)", token, source)
	}
}

func TestRequireTokenError(t *testing.T) {
	t.Setenv("SLACK_CLI_TOKEN", "")
	t.Setenv("SLACK_TOKEN", "")
	ctx := newTestContext(t)

	if _, _, err := ctx.RequireToken(); err == nil {
		t.Fatal("RequireToken() error = nil, want error")
	}
}

func TestSaveConfigPersistsFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	ctx, err := NewContext(Settings{ConfigPath: path, OutputMode: output.Human, Timeout: time.Second}, &bytes.Buffer{}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("NewContext() error = %v", err)
	}
	ctx.Config.SetToken("default", "token")

	if err := ctx.SaveConfig(); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("config not written: %v", err)
	}
}

func TestNewContextAndSaveConfigErrors(t *testing.T) {
	badPath := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(badPath, []byte("{broken"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if _, err := NewContext(Settings{ConfigPath: badPath, OutputMode: output.Human, Timeout: time.Second}, &bytes.Buffer{}, &bytes.Buffer{}); err == nil {
		t.Fatal("NewContext(invalid config) error = nil")
	}

	parentFile := filepath.Join(t.TempDir(), "parent-file")
	if err := os.WriteFile(parentFile, []byte("x"), 0o600); err != nil {
		t.Fatalf("WriteFile(parentFile) error = %v", err)
	}
	ctx := &Context{ConfigPath: filepath.Join(parentFile, "config.json"), Config: &config.File{Profiles: map[string]config.Profile{}}, Settings: Settings{Timeout: time.Second}, Output: output.New(output.Human, &bytes.Buffer{}, &bytes.Buffer{})}
	if err := ctx.SaveConfig(); err == nil {
		t.Fatal("SaveConfig(file parent) error = nil")
	}

	t.Setenv("SLACK_CLI_TOKEN", "")
	t.Setenv("SLACK_TOKEN", "")
	ctx = &Context{Config: &config.File{Profiles: map[string]config.Profile{}}, Settings: Settings{Timeout: time.Second}, Output: output.New(output.Human, &bytes.Buffer{}, &bytes.Buffer{})}
	if _, err := ctx.SlackClient(); err == nil {
		t.Fatal("SlackClient(no token) error = nil")
	}
}

func TestSlackClientUsesConfiguredBaseURLAndCaches(t *testing.T) {
	server := newSlackServer(t)
	defer server.Close()
	t.Setenv("SLACK_CLI_TOKEN", "token")
	t.Setenv("SLACK_CLI_BASE_URL", server.URL)

	ctx := newTestContext(t)
	client1, err := ctx.SlackClient()
	if err != nil {
		t.Fatalf("SlackClient() error = %v", err)
	}
	client2, err := ctx.SlackClient()
	if err != nil {
		t.Fatalf("SlackClient() second error = %v", err)
	}
	if client1 != client2 {
		t.Fatal("SlackClient() did not cache client")
	}
}

func TestResolveHelpersAndMatchers(t *testing.T) {
	server := newSlackServer(t)
	defer server.Close()
	t.Setenv("SLACK_CLI_TOKEN", "token")
	t.Setenv("SLACK_CLI_BASE_URL", server.URL)

	ctx := newTestContext(t)

	if id, err := ctx.ResolveUser("@alice"); err != nil || id != "U1" {
		t.Fatalf("ResolveUser(@alice) = (%q, %v)", id, err)
	}
	if id, err := ctx.ResolveUser("Alice Jones"); err != nil || id != "U1" {
		t.Fatalf("ResolveUser(real name) = (%q, %v)", id, err)
	}
	if ids, err := ctx.ResolveUsers([]string{"@alice", "bob"}); err != nil || len(ids) != 2 || ids[1] != "U2" {
		t.Fatalf("ResolveUsers() = (%v, %v)", ids, err)
	}
	if id, err := ctx.ResolveConversation("#general"); err != nil || id != "C1" {
		t.Fatalf("ResolveConversation(#general) = (%q, %v)", id, err)
	}
	if id, err := ctx.ResolveConversation("random"); err != nil || id != "G1" {
		t.Fatalf("ResolveConversation(random) = (%q, %v)", id, err)
	}

	if !userMatches(slackUser("U1", "alice", "Alice Jones", "aj"), "@alice") {
		t.Fatal("userMatches() false, want true")
	}
	if !conversationMatches(slackConversation("C1", "general"), "#general") {
		t.Fatal("conversationMatches() false, want true")
	}
	if normalizeRef("  @Alice ") != "@alice" {
		t.Fatalf("normalizeRef() unexpected")
	}
	if !isConversationID("C123") || isConversationID("U123") {
		t.Fatalf("isConversationID() unexpected")
	}
	if firstNonEmpty("", "x") != "x" {
		t.Fatalf("firstNonEmpty() unexpected")
	}
	if firstValue("x", "y") != "x" {
		t.Fatalf("firstValue() unexpected")
	}
}

func TestResolveErrorsForMissingUserAndConversation(t *testing.T) {
	server := newSlackServer(t)
	defer server.Close()
	t.Setenv("SLACK_CLI_TOKEN", "token")
	t.Setenv("SLACK_CLI_BASE_URL", server.URL)

	ctx := newTestContext(t)
	if _, err := ctx.ResolveUser("nobody"); err == nil {
		t.Fatal("ResolveUser() error = nil, want error")
	}
	if _, err := ctx.ResolveConversation("unknown-room"); err == nil {
		t.Fatal("ResolveConversation() error = nil, want error")
	}
}

func TestResolveRawIDsAndEmptyErrors(t *testing.T) {
	ctx := newTestContext(t)
	if id, err := ctx.ResolveUser("UABC123"); err != nil || id != "UABC123" {
		t.Fatalf("ResolveUser(raw) = (%q, %v)", id, err)
	}
	if id, err := ctx.ResolveConversation("CABC123"); err != nil || id != "CABC123" {
		t.Fatalf("ResolveConversation(raw) = (%q, %v)", id, err)
	}
	if _, err := ctx.ResolveUser("   "); err == nil {
		t.Fatal("ResolveUser(empty) error = nil")
	}
	if _, err := ctx.ResolveConversation("   "); err == nil {
		t.Fatal("ResolveConversation(empty) error = nil")
	}
}

func TestResolveAmbiguousMatchesAndPagination(t *testing.T) {
	usersCalls := 0
	conversationsCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/users.list":
			usersCalls++
			if usersCalls == 1 {
				_, _ = w.Write([]byte(`{"ok":true,"members":[{"id":"U1","name":"alex","profile":{"display_name":"first"}}],"response_metadata":{"next_cursor":"next"}}`))
				return
			}
			_, _ = w.Write([]byte(`{"ok":true,"members":[{"id":"U2","name":"alex","profile":{"display_name":"second"}}],"response_metadata":{"next_cursor":""}}`))
		case "/conversations.list":
			conversationsCalls++
			if conversationsCalls == 1 {
				_, _ = w.Write([]byte(`{"ok":true,"channels":[{"id":"C1","name":"team"}],"response_metadata":{"next_cursor":"next"}}`))
				return
			}
			_, _ = w.Write([]byte(`{"ok":true,"channels":[{"id":"G1","name":"team","is_private":true}],"response_metadata":{"next_cursor":""}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"ok":false,"error":"not_found"}`))
		}
	}))
	defer server.Close()

	t.Setenv("SLACK_CLI_TOKEN", "token")
	t.Setenv("SLACK_CLI_BASE_URL", server.URL)
	ctx := newTestContext(t)

	if _, err := ctx.ResolveUser("alex"); err == nil || err.Error() == "" {
		t.Fatal("ResolveUser(ambiguous) error = nil")
	}
	if _, err := ctx.ResolveConversation("team"); err == nil || err.Error() == "" {
		t.Fatal("ResolveConversation(ambiguous) error = nil")
	}
	if usersCalls < 2 || conversationsCalls < 2 {
		t.Fatalf("pagination not exercised: users=%d conversations=%d", usersCalls, conversationsCalls)
	}
}

func slackUser(id, name, realName, displayName string) slackapi.User {
	return slackapi.User{
		ID:       id,
		Name:     name,
		RealName: realName,
		Profile: slackapi.UserProfile{
			DisplayName: displayName,
			RealName:    realName,
		},
	}
}

func slackConversation(id, name string) slackapi.Conversation {
	return slackapi.Conversation{ID: id, Name: name, NameNormalized: name}
}
