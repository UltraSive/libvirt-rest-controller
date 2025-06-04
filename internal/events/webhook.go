package events

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

// WebhookPayload represents the structure of the JSON payload for the webhook.
type WebhookPayload struct {
	Object    string                 `json:"object"`
	NodeID    string                 `json:"node_id"`
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Data      map[string]interface{} `json:"data"`
	Timestamp string                 `json:"timestamp"`
}

// SendWebhook sends a JSON payload as a POST request to a webhook URL
// specified by an environment variable.
func SendWebhook(payload WebhookPayload) error {
	// 1. Load the webhook URL from environment variable
	webhookURL := os.Getenv("WEBHOOK_URL")
	if webhookURL == "" {
		return fmt.Errorf("WEBHOOK_URL environment variable not set")
	}

	// 2. Marshal the payload to JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	// 3. Create a new HTTP client
	// It's good practice to use a client with a timeout
	client := &http.Client{
		Timeout: 10 * time.Second, // Set a timeout for the request
	}

	// 4. Create a new HTTP POST request
	req, err := http.NewRequest("POST", webhookURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// 5. Set the Content-Type header to application/json
	req.Header.Set("Content-Type", "application/json")

	// 6. Send the request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer resp.Body.Close() // Ensure the response body is closed

	// 7. Read and check the response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("webhook returned non-2xx status code: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	fmt.Printf("Webhook successfully sent to %s. Status: %s\n", webhookURL, resp.Status)
	return nil
}