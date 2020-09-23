package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"testing"
)

func TestPipe(t *testing.T) {
	r, w, _ := os.Pipe()
	log.Println("write a")
	fmt.Fprint(w, "a")
	log.Println("write b")
	fmt.Fprint(w, "b")

	os.Pipe()

	log.Println("write done")
	_, _ = ioutil.ReadAll(r)
}
