package websocket

import "time"
import "sync"

type Socketconfig struct {
	writeWait             time.Duration
	readWait              time.Duration
	pingPeriod            time.Duration
	receivemaxMessageSize int64
	sendmessageBuffsize   int
}

var once = sync.Once{}
var scoketconf *Socketconfig

func init() {
	NewSocketconfig()
}

func NewSocketconfig() *Socketconfig {
	once.Do(func() {
		scoketconf = &Socketconfig{
			writeWait:             30 * time.Second,
			readWait:              60 * time.Second,
			pingPeriod:            (60 * time.Second * 9) / 10,
			receivemaxMessageSize: 512,
			sendmessageBuffsize:   512,
		}
	})
	return scoketconf
}
