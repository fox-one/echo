package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/fox-one/echo"
	"github.com/fox-one/pkg/lruset"
	"github.com/sirupsen/logrus"
)

// Message represents scan message
type Log struct {
	Level string `json:"level,omitempty"`
	Error string `json:"error,omitempty"`
	Msg   string `json:"msg,omitempty"`
}

func (log *Log) reset() {
	log.Level = ""
	log.Error = ""
	log.Msg = ""
}

var (
	stdout = flag.Bool("stdout", false, "output to stdout")
	stderr = flag.Bool("stderr", false, "output to stderr")
	format = flag.String("format", "text", "mixin message category")
)

func main() {
	flag.Parse()

	tokens := make(map[string]string)
	for _, level := range logrus.AllLevels {
		levelString := level.String()
		env := "scanner_token_" + levelString
		if token := os.Getenv(strings.ToUpper(env)); token != "" {
			tokens[level.String()] = token
			logrus.Infoln(levelString, "enabled")
		}
	}

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
	b := bytes.Buffer{}
	set := lruset.New(5)

	var log Log
	for s.Scan() {
		// reset log
		log.reset()

		raw := json.RawMessage(s.Bytes())
		if err := json.Unmarshal(raw, &log); err != nil {
			continue
		}

		token, ok := tokens[log.Level]
		if !ok {
			continue
		}

		// filter duplicated error logs
		if log.Error != "" {
			if set.Contains(log.Error) {
				continue
			}

			set.Add(log.Error)
		}

		category := "PLAIN_TEXT"
		data, _ := json.MarshalIndent(raw, "", "  ")
		if *format == "post" {
			b.Reset()
			b.WriteString("### [")
			b.WriteString(log.Level)
			b.WriteString("] ")
			b.WriteString(log.Msg)
			b.WriteString(" ###")
			b.WriteByte('\n')
			b.WriteByte('\n')
			b.WriteString("```json")
			b.WriteByte('\n')
			b.Write(data)
			b.WriteByte('\n')
			b.WriteString("```")

			data = b.Bytes()
			category = "PLAIN_POST"
		}

		payload := echo.Payload{
			Category: category,
			Data:     base64.StdEncoding.EncodeToString(data),
		}

		if err := echo.SendMessage(ctx, token, payload); err != nil {
			logrus.WithError(err).Errorf("send message: %s", log.Msg)
		}
	}

	if err := s.Err(); err != nil {
		logrus.WithError(err).Fatal("terminated")
	}
}
