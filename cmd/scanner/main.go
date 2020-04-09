package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"io"
	"io/ioutil"
	"os"
	"time"

	"github.com/fox-one/echo"
	"github.com/fox-one/mixin-sdk"
	"github.com/patrickmn/go-cache"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const defaultLevel = "default"

// Message represents scan message
type Message struct {
	Level string `json:"level,omitempty"`
	Error string `json:"error,omitempty"`
}

func (msg *Message) reset() {
	msg.Level = ""
	msg.Error = ""
}

var (
	stdout = flag.Bool("stdout", false, "output to stdout")
	stderr = flag.Bool("stderr", false, "output to stderr")
)

func main() {
	flag.Parse()

	setupViper()
	checkTokens()

	ctx := context.Background()

	var out io.Writer
	switch {
	case *stdout:
		out = os.Stdout
	case *stderr:
		out = os.Stderr
	default:
		out = ioutil.Discard
	}

	r := io.TeeReader(os.Stdin, out)
	s := bufio.NewScanner(r)
	c := cache.New(time.Minute, 5*time.Minute)

	var msg Message
	for s.Scan() {
		// reset msg level
		msg.reset()

		if err := json.Unmarshal(s.Bytes(), &msg); err != nil {
			continue
		}

		if msg.Level == "" {
			continue
		}

		if msg.Error != "" {
			if _, ok := c.Get(msg.Error); ok {
				continue
			}

			c.SetDefault(msg.Error, nil)
		}

		token, ok := getToken(msg.Level)
		if !ok {
			continue
		}

		data, _ := json.MarshalIndent(json.RawMessage(s.Bytes()), "", "    ")
		payload := echo.Payload{
			Category: mixin.MessageCategoryPlainText,
			Data:     base64.StdEncoding.EncodeToString(data),
		}

		for i := 0; i < 5; i++ {
			if err := echo.SendMessage(ctx, token, payload); err != nil {
				logrus.WithError(err).Error("send message")
				time.Sleep(time.Second)
				continue
			}

			break
		}

	}

	logrus.WithError(s.Err()).Infoln("terminated")
}

func setupViper() {
	viper.SetEnvPrefix("scanner_token")
	for _, level := range logrus.AllLevels {
		_ = viper.BindEnv(level.String())
	}

	_ = viper.BindEnv(defaultLevel)
}

func checkTokens() {
	for _, level := range logrus.AllLevels {
		_, ok := getToken(level.String())
		logrus.Infoln("scanner token", level.String(), ok)
	}
}

func getToken(level string) (string, bool) {
	v, err := logrus.ParseLevel(level)
	if err != nil {
		return "", false
	}

	token := viper.GetString(v.String())
	if token == "" {
		token = viper.GetString(defaultLevel)
	}

	return token, token != ""
}
