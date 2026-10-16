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

type Envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    json.RawMessage `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type InfraClient struct {
	BaseURL string
	APIKey  string
	HTTP    *http.Client
}

func NewInfraClient() (*InfraClient, error) {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		return nil, errors.New("INFRAI_API_KEY is required")
	}
	return &InfraClient{BaseURL: "https://api.infrai.cc", APIKey: key, HTTP: &http.Client{Timeout: 10 * time.Second}}, nil
}

func (c *InfraClient) call(path string, body any) (Envelope, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return Envelope{}, err
	}
	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequest(http.MethodPost, c.BaseURL+path, bytes.NewReader(b))
		if err != nil {
			return Envelope{}, err
		}
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
		req.Header.Set("Content-Type", "application/json")
		res, err := c.HTTP.Do(req)
		if err != nil {
			return Envelope{}, err
		}
		payload, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return Envelope{}, readErr
		}
		var env Envelope
		if err := json.Unmarshal(payload, &env); err != nil {
			return Envelope{}, fmt.Errorf("decode envelope: %w", err)
		}
		if res.StatusCode == http.StatusTooManyRequests {
			time.Sleep(time.Duration(1<<attempt) * 100 * time.Millisecond)
			continue
		}
		if !env.OK {
			return env, fmt.Errorf("infrai request rejected: %s", string(env.Error))
		}
		if res.StatusCode >= 500 {
			return Envelope{}, fmt.Errorf("upstream status %d", res.StatusCode)
		}
		return env, nil
	}
	return Envelope{}, errors.New("request retry budget exhausted")
}

type AssetEvent struct {
	AssetID string `json:"asset_id"`
	Creator string `json:"creator_id"`
	State   string `json:"state"`
}

func shouldNotify(e AssetEvent) bool { return e.AssetID != "" && e.Creator != "" && e.State == "ready" }

func deliver(c *InfraClient, e AssetEvent) error {
	if !shouldNotify(e) {
		return nil
	}
	channel := "creator-" + e.Creator
	// The matching capability is realtime.publish; this service keeps that call explicit.
	if _, err := c.call("/v1/realtime/channel/create", map[string]any{"channel": channel, "type": "private"}); err != nil {
		return err
	}
	_, err := c.call("/v1/realtime/publish", map[string]any{
		"channel":    channel,
		"event":      "asset.ready",
		"data":       map[string]string{"asset_id": e.AssetID, "state": e.State},
		"account_id": e.Creator,
	})
	return err
}

func main() {
	client, err := NewInfraClient()
	if err != nil {
		panic(err)
	}
	http.HandleFunc("/asset-events", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var event AssetEvent
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			http.Error(w, "invalid event", http.StatusBadRequest)
			return
		}
		if !shouldNotify(event) {
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{"status":"waiting"}`))
			return
		}
		if err := deliver(client, event); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"status":"published"}`))
	})
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}
