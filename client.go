package echo

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

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

// Get execute a get request
func Get(ctx context.Context, uri string) ([]byte, error) {
	return Execute(ctx, http.MethodGet, uri, nil)
}

// Execute execute a request
func Execute(ctx context.Context, method, uri string, body interface{}) ([]byte, error) {
	resp, err := client.R().SetContext(ctx).SetBody(body).Execute(method, uri)
	if err != nil {
		return nil, err
	}

	if resp.IsError() {
		return nil, errors.New(resp.Status())
	}

	var r struct {
		Data json.RawMessage `json:"data,omitempty"`
	}
	err = json.Unmarshal(resp.Body(), &r)
	return r.Data, err
}

// SendMessage send a message
func SendMessage(ctx context.Context, token string, payload Payload) error {
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
