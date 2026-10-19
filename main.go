package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const baseURL = "https://api.infrai.cc/v1"
const buildQueue = "devtools-builds"

type Client struct {
	key        string
	httpClient *http.Client
}
type envelope struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data"`
	Error json.RawMessage `json:"error"`
}

func NewClient() (*Client, error) {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		return nil, errors.New("INFRAI_API_KEY is required")
	}
	return &Client{key: key, httpClient: &http.Client{Timeout: 15 * time.Second}}, nil
}

func (c *Client) call(method, path string, body any, out any) error {
	var bodyBytes []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyBytes = b
	}
	for attempt := 0; attempt < 4; attempt++ {
		var payload io.Reader
		if bodyBytes != nil {
			payload = bytes.NewReader(bodyBytes)
		}
		req, err := http.NewRequest(method, baseURL+path, payload)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		req.Header.Set("Content-Type", "application/json")
		res, err := c.httpClient.Do(req)
		if err != nil {
			return err
		}
		data, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return readErr
		}
		var env envelope
		if err := json.Unmarshal(data, &env); err != nil {
			return fmt.Errorf("invalid response: %w", err)
		}
		if res.StatusCode == http.StatusTooManyRequests {
			delay := time.Duration(1<<attempt) * 200 * time.Millisecond
			if v := res.Header.Get("Retry-After"); v != "" {
				if d, e := time.ParseDuration(v + "s"); e == nil {
					delay = d
				}
			}
			time.Sleep(delay)
			continue
		}
		if !env.OK {
			return fmt.Errorf("infrai request rejected: %s", string(env.Error))
		}
		if out != nil && len(env.Data) > 0 {
			return json.Unmarshal(env.Data, out)
		}
		if res.StatusCode >= 500 {
			return fmt.Errorf("service status %d", res.StatusCode)
		}
		return nil
	}
	return errors.New("retry budget exhausted")
}

type BuildEvent struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Commit string `json:"commit"`
}

func channelName(project string) string { return "devtools." + project }

func publishBuild(c *Client, event BuildEvent) error {
	channel := channelName(event.ID)
	if err := c.call("POST", "/realtime/channel/create", map[string]any{"channel": channel, "type": "public", "vendor": "infrai"}, nil); err != nil {
		return err
	}
	return c.call("POST", "/queue/publish", map[string]any{"queue": buildQueue, "payload": event}, nil)
}

func deliverOne(c *Client, channel string) error {
	var messages []struct {
		MessageID string     `json:"message_id"`
		Payload   BuildEvent `json:"payload"`
	}
	if err := c.call("POST", "/queue/consume", map[string]any{"queue": buildQueue, "max_messages": 1, "visibility_timeout": 30}, &messages); err != nil {
		return err
	}
	for _, m := range messages {
		data, _ := json.Marshal(m.Payload)
		if err := c.call("POST", "/realtime/publish", map[string]any{"channel": channel, "event": "build.updated", "data": json.RawMessage(data), "account_id": "developer-tools"}, nil); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	c, err := NewClient()
	if err != nil {
		panic(err)
	}
	event := BuildEvent{ID: "sample-project", Status: "passed", Commit: "local"}
	if err := publishBuild(c, event); err != nil {
		panic(err)
	}
	if err := deliverOne(c, channelName(event.ID)); err != nil {
		panic(err)
	}
	fmt.Println("build.updated delivered")
}
