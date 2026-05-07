package slack

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func newSlackTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth.test":
			_, _ = w.Write([]byte(`{"ok":true,"team":"Team","user":"User"}`))
		case "/users.list":
			_, _ = w.Write([]byte(`{"ok":true,"members":[{"id":"U1","name":"alice"}]}`))
		case "/users.info":
			_, _ = w.Write([]byte(`{"ok":true,"user":{"id":"U1","name":"alice"}}`))
		case "/users.getPresence":
			_, _ = w.Write([]byte(`{"ok":true,"presence":"active","online":true}`))
		case "/conversations.list":
			_, _ = w.Write([]byte(`{"ok":true,"channels":[{"id":"C1","name":"general"}]}`))
		case "/conversations.info":
			_, _ = w.Write([]byte(`{"ok":true,"channel":{"id":"C1","name":"general","last_read":"1.0","unread_count_display":2}}`))
		case "/conversations.history":
			_, _ = w.Write([]byte(`{"ok":true,"messages":[{"ts":"2.0","text":"hello"}]}`))
		case "/conversations.replies":
			_, _ = w.Write([]byte(`{"ok":true,"messages":[{"ts":"2.1","text":"reply"}]}`))
		case "/conversations.members":
			_, _ = w.Write([]byte(`{"ok":true,"members":["U1","U2"]}`))
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
			_, _ = w.Write([]byte(`{"ok":true,"query":"q","messages":{"matches":[{"text":"hit","ts":"1.0"}],"total":1}}`))
		case "/invalid":
			_, _ = w.Write([]byte(`{"ok":false,"error":"invalid_auth"}`))
		case "/badhttp":
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte("bad gateway"))
		case "/ratelimit":
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte("rate limited"))
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(fmt.Sprintf("unknown path %s", r.URL.Path)))
		}
	}))
}

func TestNewClientDefaults(t *testing.T) {
	client := NewClient(Options{})
	if client.baseURL != defaultBaseURL {
		t.Fatalf("baseURL = %q", client.baseURL)
	}
	if client.maxRateLimitRetries != 2 {
		t.Fatalf("retries = %d", client.maxRateLimitRetries)
	}
}

func TestCallWithHTTPMethodGetSendsHeadersAndDecodes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/auth.test" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer token" {
			t.Fatalf("Authorization = %q", got)
		}
		if got := r.Header.Get("Cookie"); got != "d=abc" {
			t.Fatalf("Cookie = %q", got)
		}
		if r.URL.RawQuery != "foo=bar" {
			t.Fatalf("query = %q", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{"ok":true,"team":"T"}`))
	}))
	defer server.Close()

	client := NewClient(Options{Token: "token", Cookie: "d=abc", BaseURL: server.URL, Timeout: time.Second})
	var out map[string]any
	err := client.CallWithHTTPMethod(context.Background(), "auth.test", http.MethodGet, url.Values{"foo": []string{"bar"}}, &out)
	if err != nil {
		t.Fatalf("CallWithHTTPMethod() error = %v", err)
	}
	if out["team"] != "T" {
		t.Fatalf("decoded team = %#v", out["team"])
	}
}

func TestCallReturnsAPIError(t *testing.T) {
	server := newSlackTestServer(t)
	defer server.Close()
	client := NewClient(Options{Token: "token", BaseURL: server.URL, Timeout: time.Second})
	err := client.Call(context.Background(), "invalid", nil, &map[string]any{})
	if err == nil || !strings.Contains(err.Error(), "invalid_auth") {
		t.Fatalf("Call() error = %v, want invalid_auth", err)
	}
}

func TestCallReturnsHTTPError(t *testing.T) {
	server := newSlackTestServer(t)
	defer server.Close()
	client := NewClient(Options{Token: "token", BaseURL: server.URL, Timeout: time.Second})
	err := client.Call(context.Background(), "badhttp", nil, &map[string]any{})
	if err == nil || !strings.Contains(err.Error(), "502") {
		t.Fatalf("Call() error = %v, want HTTP error", err)
	}
}

func TestClientMethodWrappers(t *testing.T) {
	server := newSlackTestServer(t)
	defer server.Close()
	client := NewClient(Options{Token: "token", BaseURL: server.URL, Timeout: time.Second})
	ctx := context.Background()

	checks := []func() error{
		func() error { _, err := client.AuthTest(ctx); return err },
		func() error { _, err := client.UsersList(ctx, 10, ""); return err },
		func() error { _, err := client.UsersInfo(ctx, "U1"); return err },
		func() error { _, err := client.UsersGetPresence(ctx, "U1"); return err },
		func() error { _, err := client.ConversationsList(ctx, 10, "", "public_channel", false); return err },
		func() error { _, err := client.ConversationsInfo(ctx, "C1", true); return err },
		func() error { _, err := client.ConversationsHistory(ctx, "C1", 10, "", "", "", false); return err },
		func() error {
			_, err := client.ConversationsReplies(ctx, "C1", "1.0", 10, "", "", "", false)
			return err
		},
		func() error { _, err := client.ConversationsMembers(ctx, "C1", 10, ""); return err },
		func() error { _, err := client.ConversationsOpen(ctx, []string{"U1", "U2"}, true); return err },
		func() error { _, err := client.ConversationsOpen(ctx, []string{"U1"}, false); return err },
		func() error { _, err := client.ConversationsJoin(ctx, "C1"); return err },
		func() error { _, err := client.ConversationsLeave(ctx, "C1"); return err },
		func() error { _, err := client.ConversationsCreate(ctx, "new-room", false); return err },
		func() error { _, err := client.ConversationsCreate(ctx, "private-room", true); return err },
		func() error { _, err := client.ConversationsRename(ctx, "C1", "renamed"); return err },
		func() error { _, err := client.ConversationsSetTopic(ctx, "C1", "topic"); return err },
		func() error { _, err := client.ConversationsSetPurpose(ctx, "C1", "purpose"); return err },
		func() error { return client.ConversationsArchive(ctx, "C1") },
		func() error { return client.ConversationsUnarchive(ctx, "C1") },
		func() error { return client.ConversationsMark(ctx, "C1", "1.0") },
		func() error { _, err := client.ChatPostMessage(ctx, "C1", "hi", ""); return err },
		func() error { _, err := client.ChatPostMessage(ctx, "C1", "hi", "1.0"); return err },
		func() error { _, err := client.ChatPostEphemeral(ctx, "C1", "U1", "hi", ""); return err },
		func() error { _, err := client.ChatPostEphemeral(ctx, "C1", "U1", "hi", "1.0"); return err },
		func() error { _, err := client.ChatUpdate(ctx, "C1", "1.2", "updated"); return err },
		func() error { _, err := client.ChatDelete(ctx, "C1", "1.2"); return err },
		func() error { _, err := client.ChatGetPermalink(ctx, "C1", "1.2"); return err },
		func() error { _, err := client.SearchMessages(ctx, "q", 20, 1, "score", "desc", "", false); return err },
		func() error { _, err := client.SearchMessages(ctx, "q", 20, 2, "timestamp", "asc", "cursor", true); return err },
	}

	for i, check := range checks {
		if err := check(); err != nil {
			t.Fatalf("check %d error = %v", i, err)
		}
	}
}

func TestAPIErrorAndHelpers(t *testing.T) {
	err := (&APIError{Method: "m", Code: "bad", Needed: "a", Provided: "b", Warning: "w"}).Error()
	if !strings.Contains(err, "needed=a") || !strings.Contains(err, "warning=w") {
		t.Fatalf("APIError.Error() = %q", err)
	}

	params := url.Values{}
	setString(params, "a", "x")
	setString(params, "b", "")
	setInt(params, "c", 2)
	setInt(params, "d", 0)
	setBool(params, "e", true)
	setBool(params, "f", false)

	if params.Get("a") != "x" || params.Get("c") != "2" || params.Get("e") != "true" {
		t.Fatalf("helper params = %#v", params)
	}
}

func TestCallWithHTTPMethodAdditionalBranches(t *testing.T) {
	var postBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/post-default":
			body, _ := io.ReadAll(r.Body)
			postBody = string(body)
			if got := r.Header.Get("Content-Type"); got != "application/x-www-form-urlencoded" {
				t.Fatalf("Content-Type = %q", got)
			}
			_, _ = w.Write([]byte(`{"ok":true}`))
		case "/badjson":
			_, _ = w.Write([]byte(`not-json`))
		case "/ok":
			_, _ = w.Write([]byte(`{"ok":true,"value":"x"}`))
		case "/ratelimit-cancel":
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte("rate limited"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(Options{Token: "token", BaseURL: server.URL, Timeout: time.Second})
	if err := client.CallWithHTTPMethod(context.Background(), "post-default", "", url.Values{"a": []string{"1"}}, nil); err != nil {
		t.Fatalf("CallWithHTTPMethod(default POST) error = %v", err)
	}
	if postBody != "a=1" {
		t.Fatalf("POST body = %q", postBody)
	}

	if err := client.CallWithHTTPMethod(context.Background(), "badjson", http.MethodGet, nil, &map[string]any{}); err == nil {
		t.Fatal("CallWithHTTPMethod(bad envelope) error = nil")
	}
	if err := client.CallWithHTTPMethod(context.Background(), "ok", http.MethodGet, nil, make(chan int)); err == nil {
		t.Fatal("CallWithHTTPMethod(bad output target) error = nil")
	}

	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := client.Call(cancelCtx, "ratelimit-cancel", nil, &map[string]any{}); err == nil {
		t.Fatal("Call(ratelimit canceled) error = nil")
	}
}
