package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"strings"
	"unicode/utf8"

	"github.com/spf13/cast"
)

const (
	fieldNameLevel = "level"
	fieldNameError = "error"
	fieldNameMsg   = "msg"
)

// Field represents Entry Field from logrus
type Field struct {
	Key   string
	Value string
}

// Entry represents log message
type Entry struct {
	Level  string
	Error  string
	Msg    string
	Fields []Field
}

func (log *Entry) reset() {
	log.Level = ""
	log.Error = ""
	log.Msg = ""
	log.Fields = log.Fields[0:0]
}

func renderLog(log *Entry, b *bytes.Buffer) {
	b.WriteString("### [")
	b.WriteString(log.Level)
	b.WriteString("] ")
	b.WriteString(removeSurroundingQuotes(log.Msg))
	b.WriteString(" ###")
	b.WriteByte('\n')
	b.WriteByte('\n')
	b.WriteString("```yaml")
	for _, field := range log.Fields {
		b.WriteByte('\n')
		b.WriteString(field.Key)
		b.WriteString(": ")
		b.WriteString(field.Value)
	}
	b.WriteByte('\n')
	b.WriteString("```")
}

func parseLog(token []byte, log *Entry) {
	values := make(map[string]interface{})

	// parse json
	if err := json.Unmarshal(token, &values); err != nil {
		s := bufio.NewScanner(bytes.NewReader(token))
		s.Split(scanWords)

		for s.Scan() {
			if fields := strings.SplitN(s.Text(), "=", 2); len(fields) == 2 {
				values[fields[0]] = fields[1]
			}
		}
	}

	for k, v := range values {
		value := cast.ToString(v)
		// value = removeSurroundingQuotes(value)

		switch k {
		case fieldNameLevel:
			log.Level = value
		case fieldNameError:
			log.Error = value
		case fieldNameMsg:
			log.Msg = value
		}

		log.Fields = append(log.Fields, Field{
			Key:   k,
			Value: value,
		})
	}
}

func isSpace(r rune) bool {
	if r <= '\u00FF' {
		// Obvious ASCII ones: \t through \r plus space. Plus two Latin-1 oddballs.
		switch r {
		case ' ', '\t', '\n', '\v', '\f', '\r':
			return true
		case '\u0085', '\u00A0':
			return true
		}
		return false
	}
	// High-valued ones.
	if '\u2000' <= r && r <= '\u200a' {
		return true
	}
	switch r {
	case '\u1680', '\u2028', '\u2029', '\u202f', '\u205f', '\u3000':
		return true
	}
	return false
}

func scanWords(data []byte, atEOF bool) (advance int, token []byte, err error) {
	// Skip leading spaces.
	start := 0
	for width := 0; start < len(data); start += width {
		var r rune
		r, width = utf8.DecodeRune(data[start:])
		if !isSpace(r) {
			break
		}
	}
	// Scan until space, marking end of word.
	inQuote := false
	for width, i := 0, start; i < len(data); i += width {
		var r rune
		r, width = utf8.DecodeRune(data[i:])
		if r == '"' {
			inQuote = !inQuote
		}

		if isSpace(r) && !inQuote {
			return i + width, data[start:i], nil
		}
	}
	// If we're at EOF, we have a final, non-empty, non-terminated word. Return it.
	if atEOF && len(data) > start {
		return len(data), data[start:], nil
	}
	// Request more data.
	return start, nil, nil
}

func removeSurroundingQuotes(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}

	return s
}
