package main

import (
	"encoding/json"
	"testing"
)

func TestParseToken(t *testing.T) {
	token := `a="haha xixi" b=1 level=debug msg="xixixi \"hehe\" hahah"`
	log := &Log{}
	parseLog([]byte(token), log)

	v, _ := json.MarshalIndent(log, "", " ")
	t.Log(string(v))
}
