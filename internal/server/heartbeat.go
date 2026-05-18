package server

import (
	"time"
)

const (
	DefaultHeartbeatInterval  = 2 * time.Second
	DefaultStalenessThreshold = 6 * time.Second
)
