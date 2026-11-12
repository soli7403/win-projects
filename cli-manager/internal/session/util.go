package session

import (
	"encoding/json"
	"time"

	"termgo/internal/transport"
)

type localOpts = transport.LocalOptions

func timeSleep30() { time.Sleep(30 * time.Millisecond) }

type rawMsg = json.RawMessage
