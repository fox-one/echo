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
	"net/http/httputil"
	"strconv"
	"strings"
	"time"

	"github.com/fox-one/echo"
	"github.com/fox-one/mixin-sdk"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/gofrs/uuid"
	"github.com/oxtoacart/bpool"
	"github.com/spf13/viper"
)

var (
	configFile = flag.String("config", "./config.json", "config file")
	port       = flag.Int("port", 9999, "server port")
)

const (
	host = "mixin-api.zeromesh.net"
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
			var body []byte
			if req.Body != nil {
				body, _ = ioutil.ReadAll(req.Body)
				_ = req.Body.Close()
				req.Body = ioutil.NopCloser(bytes.NewReader(body))
			}

			token, _ := user.SignToken(req.Method, req.URL.String(), body, time.Minute)
			req.Header.Set("Authorization", "Bearer "+token)
			// mixin api server 屏蔽来自 proxy 的请求
			// https://github.com/golang/go/issues/38079
			// go 1.5 上线
			req.Header["X-Forwarded-For"] = nil

			req.Host = host
			req.URL.Host = host
			req.URL.Scheme = "https"
		},
	}

	svr := &http.Server{
		Addr: fmt.Sprintf(":%d", *port),
		Handler: chain(
			proxy,
			middleware.Recoverer,
			middleware.Logger,
			middleware.NewCompressor(5).Handler,
			wrapMessage(user),
		),
	}

	if err := svr.ListenAndServe(); err != nil {
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

func wrapMessage(user *mixin.User) func(handler http.Handler) http.Handler {
	pool := bpool.NewBufferPool(64)

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
				r.Header.Set("Content-Length", strconv.Itoa(b.Len()))
				r.ContentLength = int64(b.Len())
				r.Body = ioutil.NopCloser(b)
				r.URL.Path = "/messages"
			}

			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}

func chain(h http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	for idx := 0; idx < len(middlewares); idx++ {
		h = middlewares[len(middlewares)-1-idx](h)
	}

	return h
}
