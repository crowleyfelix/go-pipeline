package log

import "log"

var (
	logger Logger
)

func SetUp(l Logger) {
	logger = l
}

func init() {
	SetUp(Noop{})
}

var Fatal = log.Fatal
