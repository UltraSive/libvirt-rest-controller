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
// (Keep this struct definition as it's used internally to marshal the JSON)
type WebhookPayload struct {
	Object    string                 `json:"object"`
	NodeID    string                 `json:"node_id"`
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Data      map[string]interface{} `json:"data"`
	Message   string                 `json:"message"`
	Timestamp string                 `json:"timestamp"`
}

// SendWebhook sends a JSON payload as a POST request to a webhook URL
// specified by an environment variable.
// It now takes individual fields as arguments to build the payload.
func SendWebhook(
	id string,
	eventType string, // Renamed 'Type' to 'eventType' to avoid conflict with Go's 'type' keyword
	message string,
	data map[string]interface{},
) error {
	// 1. Load the webhook URL and NodeID from environment variables
	webhookURL := os.Getenv("WEBHOOK_URL")
	if webhookURL == "" {
		return fmt.Errorf("WEBHOOK_URL environment variable not set")
	}

	nodeID := os.Getenv("NODE_ID")
	if nodeID == "" {
		return fmt.Errorf("NODE_ID environment variable not set")
	}

	// 2. Construct the WebhookPayload from the arguments and env vars
	payload := WebhookPayload{
		Object:    "event",
		NodeID:    nodeID, // Sourced from environment variable
		ID:        id,
		Type:      eventType, // Use eventType here
		Data:      data,
		Message:   message,
		Timestamp: time.Now().Format(time.RFC3339), // Generate timestamp within the function
	}

	// 3. Marshal the payload to JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	// 4. Create a new HTTP client
	client := &http.Client{
		Timeout: 10 * time.Second, // Set a timeout for the request
	}

	// 5. Create a new HTTP POST request
	req, err := http.NewRequest("POST", webhookURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// 6. Set the Content-Type header to application/json
	req.Header.Set("Content-Type", "application/json")

	// 7. Send the request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer resp.Body.Close() // Ensure the response body is closed

	// 8. Read and check the response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("webhook returned non-2xx status code: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	fmt.Printf("Webhook successfully sent to %s. Status: %s\n", webhookURL, resp.Status)
	return nil
}
