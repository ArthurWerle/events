package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Event is a minimal representation of an enqueued job returned by the events service.
type Event struct {
	ID          uint      `json:"id"`
	JobType     string    `json:"job_type"`
	Payload     string    `json:"payload"`
	CallbackURL string    `json:"callback_url"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

// Client sends jobs to a running events service over HTTP.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// New creates a Client that talks to the events service at baseURL.
// Example: client.New("http://events-service:3000")
func New(baseURL string) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// Enqueue submits a job to the events service queue.
// The events service will call callbackURL via GET with job_type and payload as query params.
func (c *Client) Enqueue(jobType, payload, callbackURL string) (Event, error) {
	body, err := json.Marshal(map[string]string{
		"job_type":     jobType,
		"payload":      payload,
		"callback_url": callbackURL,
	})
	if err != nil {
		return Event{}, fmt.Errorf("events client: marshal: %w", err)
	}

	resp, err := c.httpClient.Post(c.baseURL+"/api/events", "application/json", bytes.NewReader(body))
	if err != nil {
		return Event{}, fmt.Errorf("events client: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return Event{}, fmt.Errorf("events client: unexpected status %d", resp.StatusCode)
	}

	var event Event
	if err := json.NewDecoder(resp.Body).Decode(&event); err != nil {
		return Event{}, fmt.Errorf("events client: decode response: %w", err)
	}

	return event, nil
}
