package websocket

import (
	"DaiZong/config"
	"DaiZong/queue"
	"DaiZong/util"
	"encoding/json"
	"github.com/Sirupsen/logrus"
	"net/http"
)

var (
	log = logrus.New()
	SQS *queue.Queue
)

type Handler struct {
	Config   *Socketconfig
	Upgrader *Upgrader
	dis      *Dispatcher
}

func init() {

	//SQS, _ = queue.NewQueue(config.Conf.GetYamlValue("high_priority_queue_name"), config.Conf.GetYamlValue("region"))
	SQS, _ = queue.NewQueue(config.Conf.GetYamlValue("common_queue_name"), config.Conf.GetYamlValue("region"))
}

func NewHandler() *Handler {
	u := &Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	dis := NewDispatcher()
	go dis.Run()
	return &Handler{
		Config:   NewSocketconfig(),
		Upgrader: u,
		dis:      dis,
	}
}

func (h *Handler) HandleRequest(w http.ResponseWriter, r *http.Request) {
	conn, err := h.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.WithFields(logrus.Fields{"conn": conn, "error": err}).Info("upgrade websocket operation")
	}
	session := &Session{
		Request: r,
		conn:    conn,
		output:  make(chan *MessageInfo, h.Config.sendmessageBuffsize),
		disp:    h.dis,
		hd:      h,
	}
	h.dis.Register <- session
	go session.WritePump()
	session.readPump()
}

//store high_queue_name/common_queue_name
func (h *Handler) handlerMessage(s *Session, message []byte) {
	if message != nil {
		var up_message UpStreamData
		json.Unmarshal(message, &up_message)
		if up_message.Types == REGISTERMESSAGE {
			user_info := UserInfo{UpStreamData: up_message, Status: true, Servername: util.IpAddress()}
			user_message ,_ :=json.Marshal(user_info)
			deviceid,_ :=up_message.Data["deviceid"].(string)
			h.dis.Sessions[s] =deviceid
			SQS.SendMessage(string(user_message[:len(user_message)]))
		} else if up_message.Types == ACKMESSAGE {
			SQS.SendMessage(string(message[:len(message)]))
		}
	}
}

func (h *Handler) handlerError(s *Session, err error) {
	deviceid := s.disp.Sessions[s]
	log.WithFields(logrus.Fields{"session": s, "error": err,"deviceid":deviceid}).Info("write message to client")
}
