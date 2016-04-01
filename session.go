package websocket

import (
	"encoding/json"
	"errors"
	"github.com/Sirupsen/logrus"
	"net/http"
	"time"
)

type Session struct {
	Request *http.Request
	conn    *Conn
	output  chan *MessageInfo
	disp    *Dispatcher
	hd      *Handler
}

func (s *Session) WriteMessage(message *MessageInfo) {
	select {
	case s.disp.Broadcast <- message:
	default:
		s.hd.handlerError(s, errors.New("Message buffer full"))
	}
}

//此处有问题，需要调试
func (s *Session) WriteRaw(message *MessageInfo) error {
	s.conn.SetWriteDeadline(time.Now().Add(s.hd.Config.writeWait))
	b := message.Ws_type
	err := s.conn.WriteMessage(b, message.Message)
	if err != nil {
		return err
	}
	if message.Ws_type == CloseMessage {
		err := s.conn.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Session) WritePump() {
	ticker := time.NewTicker(s.hd.Config.pingPeriod)
	defer ticker.Stop()
	loop:
	for {
		select {
		//case msg, ok := <-s.output:
		case msg, ok := <-s.disp.Broadcast:
			if !ok {
				s.Close()
				break loop
			}

			if err := s.WriteRaw(msg); err != nil {
				s.hd.handlerError(s, err)
				break loop
			}
		case <-ticker.C:
			s.ping()
		}
	}

}

//读取通道的信息
func (s *Session) readPump() {
	defer func() {
		if err := recover(); err != nil {
			//s.conn.Close()
			//s.disp.removeSession(s)
			log.WithFields(logrus.Fields{"session": s, "err": err}).Info("read data fail")
		}
	}()

	s.conn.SetReadLimit(s.hd.Config.receivemaxMessageSize)
	s.conn.SetReadDeadline(time.Now().Add(s.hd.Config.readWait))
	s.conn.SetPongHandler(func(string) error {
		err := s.conn.SetReadDeadline(time.Now().Add(s.hd.Config.readWait))
		if err != nil {
			log.WithFields(logrus.Fields{"session": s, "err": err}).Info("setpong fail")
		}
		return nil
	})

	for {

		t, message, err := s.conn.ReadMessage()

		if err != nil {
			s.hd.handlerError(s, err)
		}
		if t == TextMessage {
			log.WithFields(logrus.Fields{"ws_type": t, "message": string(message[:len(message)])}).Info("read websocket info")
			s.hd.handlerMessage(s, message)
		}
	}
}

func (s *Session) ping() {
	defer func() {
		if err :=recover(); err !=nil{
			log.WithFields(logrus.Fields{"error":err}).Info(" crash ")
		}
	}()
	log.WithFields(logrus.Fields{"ping": "ping"}).Info("send ping cmd")
	err := s.WriteRaw(&MessageInfo{Ws_type: PingMessage, Message: []byte("ping test")})
	var deviceid string
	if err != nil {
		log.WithFields(logrus.Fields{"error": err}).Info("send ping cmd ")
		for k, v := range s.disp.Sessions {
			if s == k {
				deviceid = v
			}
		}
		if deviceid == "" || deviceid == "true" {
			return
		}
		m := make(map[string]interface{})
		m["deviceid"] = deviceid
		u := UserInfo{UpStreamData: UpStreamData{Types: ONOFFLINE, Data:m}, Status: false}
		rs, _ := json.Marshal(u)
		SQS.SendMessage(string(rs))
	}else {
		for k, v := range s.disp.Sessions {
			if s == k {
				deviceid = v
			}
		}
		if deviceid == "" || deviceid == "true" {
			return
		}
		m := make(map[string]interface{})
		m["deviceid"] = deviceid
		u := UserInfo{UpStreamData: UpStreamData{Types: ONOFFLINE, Data:m}, Status: true}
		rs, _ := json.Marshal(u)
		SQS.SendMessage(string(rs))

	}
}

func (s *Session) Write(msg []byte) {
	s.WriteMessage(&MessageInfo{Ws_type: TextMessage, Message: msg})
}

func (s *Session) Close() {
	s.WriteRaw(&MessageInfo{Ws_type: CloseMessage, Message: []byte{}})
}
