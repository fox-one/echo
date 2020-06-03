package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/fox-one/echo"
	"github.com/fox-one/mixin-sdk"
	"github.com/go-chi/render"
	"github.com/gofrs/uuid"
	"github.com/oxtoacart/bpool"
	"github.com/spf13/viper"
)

var (
	configFile = flag.String("config", "./config.json", "config file")
	port       = flag.Int("port", 9999, "server port")
	pool       = bpool.NewBufferPool(64)
)

const (
	host = "api.mixin.one"
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

	proxy := &httputil.ReverseProxy{
		BufferPool: bpool.NewBytePool(64, 1024*8),
		Director: func(req *http.Request) {
			req.URL.Host = host
			req.URL.Scheme = "https"

			var body []byte
			if req.Body != nil {
				b := pool.Get()
				r := io.TeeReader(req.Body, b)
				body, _ = ioutil.ReadAll(r)
				_ = req.Body.Close()
				req.Body = ioutil.NopCloser(b)
			}

			uri := extractUri(req.URL)
			token, _ := user.SignToken(req.Method, uri, body, time.Minute)
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header["X-Forward-X"] = nil
		},
	}

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", *port),
		Handler: wrapMessage(user)(proxy),
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

func extractUri(u *url.URL) string {
	s := u.String()
	idx := strings.Index(s, u.Path)
	return s[idx:]
}

func wrapMessage(user *mixin.User) func(handler http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost && r.URL.Path == "/message" {
				conversationID, err := extractConversationID(r, user)
				if err != nil {
					// token invalid
					render.Status(r, http.StatusOK)
					render.JSON(w, r, render.M{
						"status":      202,
						"code":        401,
						"description": "Unauthorized, maybe invalid token.",
					})

					return
				}

				var msg mixin.MessageRequest
				_ = json.NewDecoder(r.Body).Decode(&msg)
				_ = r.Body.Close()

				msg.ConversationID = conversationID
				if msg.MessageID == "" {
					msg.MessageID = uuid.Must(uuid.NewV4()).String()
				}

				b := pool.Get()
				defer pool.Put(b)
				_ = json.NewEncoder(b).Encode(msg)
				r.Body = ioutil.NopCloser(b)

				r.URL.Path = "/messages"
			}

			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}
