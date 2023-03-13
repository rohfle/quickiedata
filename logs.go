package quickiedata

import (
	"io"
	"log"
	"os"
)

var Log = log.Default()
var DebugLog = log.New(io.Discard, "debug: ", log.LstdFlags)

func EnableDebugLogs() {
	DebugLog.SetOutput(os.Stdout)
}

func DisableDebugLogs() {
	DebugLog.SetOutput(io.Discard)
}
