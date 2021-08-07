package logger

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"time"
)

// Simple logger wrapper
type SimpleLogger struct {
	*log.Logger
	w       io.Writer
	verbose bool
}

func (l *SimpleLogger) Verbose(use bool) {
	toUse := use && !l.verbose
	toSilent := !use && l.verbose
	if toUse {
		l.SetOutput(l.w)
	} else if toSilent {
		l.SetOutput(sw)
	}
}

func (l *SimpleLogger) Copy() *SimpleLogger {
	return New(l.w, l.Prefix(), l.verbose)
}

func (l *SimpleLogger) AppendPrefix(prefix string) {
	l.SetPrefix(fmt.Sprintf("%s%s", l.Prefix(), prefix))
}

func (l *SimpleLogger) WithPrefix(prefix string) *SimpleLogger {
	c := l.Copy()
	c.AppendPrefix(prefix)
	return c
}

type silentWriter struct{}

func (w *silentWriter) Write(p []byte) (n int, err error) {
	return 0, nil
}

var sw = &silentWriter{}

func New(out io.Writer, prefix string, verbose bool) *SimpleLogger {
	l := &SimpleLogger{log.New(sw, prefix, log.Ldate|log.Ltime), out, verbose}
	if verbose {
		l.SetOutput(out)
	}
	return l
}

var Rando *rand.Rand

func init() {
	Rando = NewRand()
}

func ByteToUpperCase(b byte) byte {
	if b > 96 && b < 123 {
		return b - 32
	}
	return b
}

func ByteToLowerCase(b byte) byte {
	if b > 64 && b < 91 {
		return b + 32
	}
	return b
}

func ToCamelCase(name string) string {
	return fmt.Sprintf("%c%s", ByteToLowerCase(name[0]), name[1:])
}

func ToPascalCase(name string) string {
	return fmt.Sprintf("%c%s", ByteToUpperCase(name[0]), name[1:])
}

func GetOr(obj interface{}, otherwise func() interface{}) interface{} {
	if obj != nil {
		return obj
	}
	return otherwise()
}

func ConditionalPick(cond bool, onTrue interface{}, onFalse interface{}) interface{} {
	if cond {
		return onTrue
	} else {
		return onFalse
	}
}

func ConditionalGet(cond bool, onTrue func() interface{}, onFalse func() interface{}) interface{} {
	if cond {
		return onTrue()
	} else {
		return onFalse()
	}
}

func SliceToSet(slice []interface{}) map[interface{}]bool {
	m := make(map[interface{}]bool)
	for _, v := range slice {
		m[v] = true
	}
	return m
}

func CopySet(set map[interface{}]bool) map[interface{}]bool {
	c := make(map[interface{}]bool)
	for k, v := range set {
		c[k] = v
	}
	return c
}

func SetIntersections(l map[interface{}]bool, r map[interface{}]bool) map[interface{}]bool {
	lCopy := CopySet(l)
	rCopy := CopySet(r)
	for k := range lCopy {
		if rCopy[k] {
			lCopy[k] = false
			rCopy[k] = false
		} else {
			rCopy[k] = true
		}
	}
	return rCopy
}

func StringArrayToInterfaceArray(arr []string) []interface{} {
	res := make([]interface{}, len(arr))
	for i := range arr {
		res[i] = arr[i]
	}
	return res
}

func NewRand() *rand.Rand {
	return rand.New(rand.NewSource(time.Now().Unix()))
}

func LogError(logger *SimpleLogger, fnName string, err error) {
	if err != nil {
		logger.Printf("error happened at %s due to %s", fnName, err.Error())
	}
}
