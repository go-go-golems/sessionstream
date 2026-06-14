package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	settings, err := parseArgs(os.Args[1:])
	if err != nil {
		panic(err)
	}
	if err := run(settings); err != nil {
		panic(err)
	}
}

type settings struct {
	PostAddr string
	WSAddr   string
}

func parseArgs(args []string) (settings, error) {
	settings := settings{PostAddr: "127.0.0.1:18789", WSAddr: "127.0.0.1:18789"}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--addr":
			if i+1 >= len(args) {
				return settings, fmt.Errorf("--addr requires a value")
			}
			settings.PostAddr = args[i+1]
			settings.WSAddr = args[i+1]
			i++
		case "--post-addr":
			if i+1 >= len(args) {
				return settings, fmt.Errorf("--post-addr requires a value")
			}
			settings.PostAddr = args[i+1]
			i++
		case "--ws-addr":
			if i+1 >= len(args) {
				return settings, fmt.Errorf("--ws-addr requires a value")
			}
			settings.WSAddr = args[i+1]
			i++
		default:
			return settings, fmt.Errorf("unknown argument %q", args[i])
		}
	}
	return settings, nil
}

func run(settings settings) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	postBaseURL := "http://" + settings.PostAddr
	wsBaseURL := "http://" + settings.WSAddr
	for {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, postBaseURL+"/healthz", nil)
		resp, err := http.DefaultClient.Do(req)
		if err == nil && resp.Body != nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}
		if err == nil && resp.StatusCode == http.StatusOK {
			break
		}
		if ctx.Err() != nil {
			return fmt.Errorf("server did not become healthy: %w", ctx.Err())
		}
		time.Sleep(100 * time.Millisecond)
	}

	for {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, wsBaseURL+"/healthz", nil)
		resp, err := http.DefaultClient.Do(req)
		if err == nil && resp.Body != nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}
		if err == nil && resp.StatusCode == http.StatusOK {
			break
		}
		if ctx.Err() != nil {
			return fmt.Errorf("websocket server did not become healthy: %w", ctx.Err())
		}
		time.Sleep(100 * time.Millisecond)
	}

	wsURL := "ws://" + settings.WSAddr + "/ws"
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return fmt.Errorf("dial websocket: %w", err)
	}
	defer func() { _ = conn.Close() }()
	if err := conn.WriteJSON(map[string]any{"subscribe": map[string]any{"sessionId": "demo", "sinceSnapshotOrdinal": "0"}}); err != nil {
		return fmt.Errorf("subscribe: %w", err)
	}

	body, _ := json.Marshal(map[string]string{"sessionId": "demo", "prompt": "smoke test prompt"})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, postBaseURL+"/api/chat", bytes.NewReader(body))
	req.Header.Set("content-type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("post chat: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("post chat status %d: %s", resp.StatusCode, respBody)
	}

	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		_ = conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		_, msg, err := conn.ReadMessage()
		if err != nil {
			continue
		}
		text := string(msg)
		if strings.Contains(text, "ChatAssistantFinished") && strings.Contains(text, "Fake backend answer") {
			fmt.Println("ok: received assistant completion over websocket")
			return nil
		}
	}
	return fmt.Errorf("timed out waiting for assistant completion")
}
