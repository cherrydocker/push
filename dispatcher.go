package websocket

import "github.com/Sirupsen/logrus"

type Dispatcher struct {
	Sessions   map[*Session]string
	Broadcast  chan *MessageInfo
	Register   chan *Session
	Unregister chan *Session
	Exit       chan bool
}

var Dp Dispatcher = Dispatcher{
	Sessions:   make(map[*Session]string),
	Broadcast:  make(chan *MessageInfo),
	Register:   make(chan *Session),
	Unregister: make(chan *Session),
	Exit:       make(chan bool),
}

func NewDispatcher() *Dispatcher {
	return &Dp
}

func (dispatcher *Dispatcher) Run() {
	defer func() {
		if err := recover(); err != nil {
			log.WithFields(logrus.Fields{"err": err})
		}
	}()
	for {
		select {
		case c := <-dispatcher.Register:
			dispatcher.Sessions[c] = "true"
		case c := <-dispatcher.Unregister:
			if _, ok := dispatcher.Sessions[c]; ok {
				dispatcher.removeSession(c)
			}
		case <-dispatcher.Exit:
			for c := range dispatcher.Sessions {
				dispatcher.removeSession(c)
			}

		}
	}
}

func (dispatcher *Dispatcher) removeSession(session *Session) {
	delete(dispatcher.Sessions, session)
	//close(session.output)
	close(session.disp.Broadcast)
}


