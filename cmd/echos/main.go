package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/fox-one/echo"
	"github.com/fox-one/mixin-sdk"
	"github.com/fox-one/pkg/uuid"
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

	var handler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var resp json.RawMessage
		if err := user.Request(r.Context(), r.Method, r.URL.String(), r.Body, &resp); err != nil {
			render.Status(r, http.StatusBadRequest)
			render.DefaultResponder(w, r, render.M{
				"error": err.Error(),
			})
		} else {
			render.Status(r, http.StatusOK)
			render.DefaultResponder(w, r, render.M{
				"data": resp,
			})
		}
	})

	handler = handleMessages(user)(handler)
	handler = middleware.Logger(handler)
	handler = middleware.Heartbeat("/hc")(handler)
	handler = cors.AllowAll().Handler(handler)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", *port),
		Handler: handler,
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
	}

	return "", errors.New("invalid authorization token")
}

func handleMessages(user *mixin.User) func(handler http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost && r.URL.Path == "/messages" {
				conversationID, err := extractConversationID(r, user)
				if err != nil {
					render.Status(r, http.StatusUnauthorized)
					render.DefaultResponder(w, r, render.M{
						"error": err.Error(),
					})
					return
				}

				var msg mixin.MessageRequest
				_ = json.NewDecoder(r.Body).Decode(&msg)
				_ = r.Body.Close()

				msg.ConversationID = conversationID
				if msg.MessageID == "" {
					msg.MessageID = uuid.New()
				}

				body, _ := json.Marshal(msg)
				r.Body = ioutil.NopCloser(bytes.NewBuffer(body))
			}

			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}
