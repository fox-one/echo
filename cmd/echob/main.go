package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"log"
	"time"

	"github.com/fox-one/echo"
	"github.com/fox-one/mixin-sdk-go"
	"github.com/fox-one/pkg/uuid"
	"github.com/spf13/viper"
)

var (
	configFile = flag.String("config", "./config.json", "config file")
)

func main() {
	flag.Parse()

	viper.SetConfigType("json")
	viper.SetConfigFile(*configFile)
	if err := viper.ReadInConfig(); err != nil {
		log.Fatal(err)
	}

	var (
		clientID   = viper.GetString("client_id")
		sessionID  = viper.GetString("session_id")
		privateKey = viper.GetString("private_key")
	)

	client, err := mixin.NewFromKeystore(&mixin.Keystore{
		ClientID:   clientID,
		SessionID:  sessionID,
		PrivateKey: privateKey,
	})
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	for {
		if err := client.LoopBlaze(ctx, handler{
			client:    client,
			sessionID: sessionID,
		}); err != nil {
			log.Println("LoopBlaze", err)
		}

		time.Sleep(time.Second)
	}
}

type handler struct {
	client    *mixin.Client
	sessionID string
}

func (h handler) OnAckReceipt(_ context.Context, _ *mixin.MessageView, _ string) error {
	return nil
}

func (h handler) OnMessage(ctx context.Context, msg *mixin.MessageView, userID string) error {
	data, err := base64.StdEncoding.DecodeString(msg.Data)
	if err != nil {
		data, err = base64.URLEncoding.DecodeString(msg.Data)
	}

	if err != nil {
		return nil
	}

	if msg.Category != mixin.MessageCategorySystemConversation {
		var raw json.RawMessage
		if err := json.Unmarshal(data, &raw); err == nil {
			data, _ = json.MarshalIndent(raw, "", "    ")
		}

		return h.client.SendMessage(ctx, &mixin.MessageRequest{
			ConversationID: msg.ConversationID,
			MessageID:      uuid.Modify(msg.MessageID, "reply"),
			Category:       mixin.MessageCategoryPlainText,
			Data:           base64.StdEncoding.EncodeToString(data),
		})
	}

	var payload struct {
		Action        string `json:"action,omitempty"`
		UserID        string `json:"user_id,omitempty"`
		ParticipantID string `json:"participant_id,omitempty"`
	}
	_ = json.Unmarshal(data, &payload)

	if payload.Action != mixin.ParticipantActionAdd || payload.ParticipantID != userID {
		return nil
	}

	token, err := echo.SignToken(payload.UserID, h.sessionID, msg.ConversationID)
	if err != nil {
		log.Println("sign token", err)
		return nil
	}

	return h.client.SendMessage(ctx, &mixin.MessageRequest{
		ConversationID: msg.ConversationID,
		RecipientID:    payload.UserID,
		MessageID:      uuid.Modify(msg.MessageID, "echo token"),
		Category:       mixin.MessageCategoryPlainText,
		Data:           base64.StdEncoding.EncodeToString([]byte(token)),
	})
}
