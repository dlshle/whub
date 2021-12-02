package utils

import (
	"math/rand"
	"strconv"
	"time"
)

var randomGenerator = rand.New(rand.NewSource(time.Now().UnixNano()))

func GenStringId() string {
	return strconv.FormatInt(randomGenerator.Int63n(time.Now().Unix())    , 16)
}
