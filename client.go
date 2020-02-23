package echo

import (
	"context"
	"errors"

	"github.com/go-resty/resty/v2"
)

const hostURL = "https://echo.yiplee.com"

var client = resty.New().
	SetHostURL(hostURL).
	SetHeader("Content-Type", "application/json")

// Payload represent message content
// leave RecipientID empty to broadcast
type Payload struct {
	MessageID   string `json:"message_id,omitempty"`
	RecipientID string `json:"recipient_id,omitempty"`
	Category    string `json:"category,omitempty"`
	Data        string `json:"data,omitempty"`
}

// Send Message
func Send(ctx context.Context, token string, payload Payload) error {
	req := client.R().SetContext(ctx)
	resp, err := req.SetAuthToken(token).SetBody(payload).Post("/message")
	if err != nil {
		return err
	}

	if resp.IsError() {
		return errors.New(resp.Status())
	}

	return nil
}
