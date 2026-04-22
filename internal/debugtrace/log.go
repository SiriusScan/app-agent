package debugtrace

import (
	"encoding/json"
	"os"
	"sync"
	"time"
)

const (
	logPath   = "/Users/oz/Projects/Sirius-Project/Sirius/.cursor/debug-c030d7.log"
	sessionID = "c030d7"
)

var mu sync.Mutex

func Log(runID, hypothesisID, location, message string, data map[string]interface{}) {
	entry := map[string]interface{}{
		"sessionId":    sessionID,
		"runId":        runID,
		"hypothesisId": hypothesisID,
		"location":     location,
		"message":      message,
		"data":         data,
		"timestamp":    time.Now().UnixMilli(),
	}
	line, err := json.Marshal(entry)
	if err != nil {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.Write(append(line, '\n'))
}
