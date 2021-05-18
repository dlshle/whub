package utils

import (
	"fmt"
	"strings"
)

type JsonBuilder struct {
	jsonMap map[string]string
	jsonString string
	changedSinceLastBuilt bool
}

type IJsonBuilder interface {
	Put(key string, value string) IJsonBuilder
	Build() string
}

func NewJsonBuilder() IJsonBuilder {
	return &JsonBuilder{jsonMap: make(map[string]string), jsonString: "", changedSinceLastBuilt: false}
}

func (b *JsonBuilder) Put(key string, value string) IJsonBuilder {
	b.changedSinceLastBuilt = true
	b.jsonMap[key] = value
	return b
}

func (b *JsonBuilder) Build() string {
	if b.jsonString != "" && !b.changedSinceLastBuilt {
		return b.jsonString
	}
	sb := strings.Builder{}
	c := 0
	l := len(b.jsonMap)
	sb.WriteByte('{')
	for k, v := range b.jsonMap {
		if c == l - 1 {
			sb.WriteString(fmt.Sprintf("%s:%s", k, v))
		} else {
			sb.WriteString(fmt.Sprintf("%s:%s,", k, v))
		}
		c++
	}
	sb.WriteByte('}')
	b.jsonString = sb.String()
	b.changedSinceLastBuilt = false
	return b.jsonString
}


func Quote(value string) string {
	return fmt.Sprintf("\"%s\"", value)
}

func Bracket(value string) string {
	return fmt.Sprintf("[%s]", value)
}

func BracketWith(values []string) string {
	sb := strings.Builder{}
	l := len(values)
	sb.WriteByte('[')
	for i, v := range values {
		sb.WriteString(v)
		if i != l - 1 {
			sb.WriteByte(',')
		}
	}
	sb.WriteByte(']')
	return sb.String()
}

func BracketStrings(values []string) string {
	quotedValues := make([]string, len(values))
	for i, v := range values {
		quotedValues[i] = Quote(v)
	}
	return BracketWith(quotedValues)
}