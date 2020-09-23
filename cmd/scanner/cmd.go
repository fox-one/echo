package main

import (
	"bufio"
	"encoding/json"
	"strings"
)

func parseCmd(cmd string) ([]string, bool) {
	var args []string
	if err := json.Unmarshal([]byte(cmd), &args); err != nil {
		s := bufio.NewScanner(strings.NewReader(cmd))
		s.Split(scanWords)

		for s.Scan() {
			arg := s.Text()
			args = append(args, arg)
		}
	}

	return args, len(args) >= 1
}
