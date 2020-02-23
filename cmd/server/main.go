package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/fox-one/echo"
	"github.com/fox-one/mixin-sdk"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/rs/cors"
	"github.com/spf13/viper"
)

var (
	configFile = flag.String("config", "./config.json", "config file")
	port       = flag.Int("port", 9999, "server port")
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

	r := chi.NewRouter()
	r.Use(cors.AllowAll().Handler)
	r.Use(middleware.Heartbeat("/hc"))
	r.Use(middleware.Logger)
	r.Use(Limit())

	r.Post("/message", func(w http.ResponseWriter, r *http.Request) {
		conversationID, err := extractConversationID(r, user)
		if err != nil {
			render.Status(r, http.StatusUnauthorized)
			render.DefaultResponder(w, r, render.M{
				"error": err.Error(),
			})
			return
		}

		var msg mixin.MessageRequest
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			render.Status(r, http.StatusBadRequest)
			render.DefaultResponder(w, r, render.M{
				"error": err.Error(),
			})
			return
		}

		msg.ConversationID = conversationID
		if err := user.SendMessage(r.Context(), &msg); err != nil {
			render.Status(r, http.StatusBadRequest)
			render.DefaultResponder(w, r, render.M{
				"error": err.Error(),
			})
			return
		}

		render.Status(r, http.StatusOK)
		render.DefaultResponder(w, r, render.M{})
	})

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", *port),
		Handler: r,
	}

	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func extractConversationID(r *http.Request, user *mixin.User) (string, error) {
	token := r.Header.Get("Authorization")
	token = strings.TrimPrefix(token, "Bearer ")
	if id, err := echo.ParseToken(token, user.SessionID); err == nil {
		return id, nil
	} else {
		return "", errors.New("invalid authorization token")
	}
}
