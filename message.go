package websocket


const (
	REGISTERMESSAGE     = 1000
	ACKMESSAGE          = 1001
	CONTENT_INFO_DETAIL = 1002
	UNREGISTERMESSAGE   = 1003
	ONOFFLINE           = 1004
)

type UpStreamData struct {
	Types int               `json:"type"`
	Data  map[string]interface{} `json:"data"`
}

type DownStreamData struct {
	Msgid string `json:"msgid"`
	Deviceid string `json:"deviceid"`
	Ack   bool `json:"ack"`
	Data  interface{} `json:"data"`
}

type MessageInfo struct {
	Ws_type int      `json:"ws_type"`
	//Conn    *Session `json:"session"`
	//Conn    *Session `json:"conn"`
	Message []byte   `json:"message"`
}

//register message format
type RegisterInfo struct {
	Deviceid string `json:"deviceid"`
	Username string `json:"username"`
	Session  string `json:"session"`
}

//app reply message format
type AckInfo struct {
	Msgid string
}

//store  register info  to  sqs
type UserInfo struct {
	UpStreamData
	//Conn       net.Conn
	//Conn *Session
	Status     bool
	Servername string
}

type PushData struct {
	DownStreamData
	Conn *Session
}

type Argus struct {
	Types string `json:"type"`
	Data map[string]interface{} `json:"data"`
}
