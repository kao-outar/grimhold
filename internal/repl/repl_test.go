package repl

import (
	"fmt"
	"io"
	"testing"
)

func TestWaitAmbushInterruptsBeforeLine(t *testing.T) {
	pr, pw := io.Pipe()
	defer pw.Close()
	reader := NewLineReader(pr)
	tick := make(chan struct{}, 1)
	tick <- struct{}{}

	line, ambush, eof, err := reader.Wait(tick, func() (string, bool) {
		return "EMBUSCADE", true
	})
	if err != nil || eof || line != "" || ambush != "EMBUSCADE" {
		t.Fatalf("line=%q ambush=%q eof=%v err=%v", line, ambush, eof, err)
	}
}

func TestWaitReadsLine(t *testing.T) {
	pr, pw := io.Pipe()
	reader := NewLineReader(pr)
	go func() {
		fmt.Fprintln(pw, "look")
		pw.Close()
	}()

	line, ambush, eof, err := reader.Wait(nil, nil)
	if err != nil || eof || ambush != "" || line != "look" {
		t.Fatalf("line=%q ambush=%q eof=%v err=%v", line, ambush, eof, err)
	}
}
