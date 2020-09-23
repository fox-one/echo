package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/bluele/gcache"
	"github.com/fox-one/echo"
	"github.com/sirupsen/logrus"
)

var (
	stdout = flag.Bool("stdout", false, "output to stdout")
	stderr = flag.Bool("stderr", false, "output to stderr")
	cmd    = flag.String("cmd", "", "execute shell command as input")
	// deprecated
	format = flag.String("format", "text", "deprecated")

	ctx = context.Background()
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

	var input io.Reader = os.Stdin

	var args []string
	if _ = json.Unmarshal([]byte(*cmd), &args); len(args) > 0 {
		pr, pw, err := os.Pipe()
		if err != nil {
			logrus.Panicln("os.Pipe", err)
		}

		go func() {
			defer pr.Close()
			defer pw.Close()

			if err := runCmd(pw, args[0], args[1:]...); err != nil {
				logrus.WithError(err).Errorln("cmd exist")
			}
		}()

		input = pr
	}

	if *stdout {
		input = io.TeeReader(input, os.Stdout)
	} else if *stderr {
		input = io.TeeReader(input, os.Stderr)
	}

	s := bufio.NewScanner(input)
	b := &bytes.Buffer{}
	limiters := gcache.New(5).LRU().Build()

	var log Entry
	for s.Scan() {
		// reset log
		log.reset()
		b.Reset()

		parseLog(s.Bytes(), &log)
		token, ok := tokens[log.Level]
		if !ok {
			continue
		}

		// filter duplicated error logs
		if log.Error != "" {
			if _, err := limiters.Get(log.Error); err != gcache.KeyNotFoundError {
				continue
			}

			_ = limiters.SetWithExpire(log.Error, struct{}{}, time.Minute)
		}

		renderLog(&log, b)

		payload := echo.Payload{
			Category: "PLAIN_POST",
			Data:     base64.StdEncoding.EncodeToString(b.Bytes()),
		}

		if err := echo.SendMessage(ctx, token, payload); err != nil {
			logrus.WithError(err).Errorf("send message: %s", log.Msg)
		}
	}
}

func runCmd(w io.Writer, name string, args ...string) error {
	for {
		c := exec.Command(name, args...)
		c.Env = os.Environ()
		c.Stderr = w
		c.Stdout = w

		if err := c.Start(); err != nil {
			return err
		}

		// 如果程序非正常退出，等待 1s，继续执行
		// 否则结束
		if err := c.Wait(); err == nil {
			return nil
		}

		time.Sleep(time.Second)
		logrus.WithField("args", strings.Join(args, " ")).Infof("Restart %s", name)
	}
}
