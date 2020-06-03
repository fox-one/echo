package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"log"
	"time"

	"github.com/fox-one/echo"
	"github.com/fox-one/mixin-sdk"
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

	user, err := mixin.NewUser(clientID, sessionID, privateKey)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	blaze := mixin.NewBlazeClient(user)
	handler := &handler{user: user}

	for {
		if err := blaze.Loop(ctx, handler); err != nil {
			log.Println("Loop", err)
		}

		time.Sleep(time.Second)
	}
}

type handler struct {
	user *mixin.User
}

func (h handler) OnAckReceipt(ctx context.Context, msg *mixin.MessageView, userID string) error {
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

		return h.user.SendMessage(ctx, &mixin.MessageRequest{
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

	token, err := echo.SignToken(payload.UserID, h.user.SessionID, msg.ConversationID)
	if err != nil {
		log.Println("sign token", err)
		return nil
	}

	return h.user.SendMessage(ctx, &mixin.MessageRequest{
		ConversationID: msg.ConversationID,
		RecipientID:    payload.UserID,
		MessageID:      uuid.Modify(msg.MessageID, "echo token"),
		Category:       mixin.MessageCategoryPlainText,
		Data:           base64.StdEncoding.EncodeToString([]byte(token)),
	})
}
