package server

import (
	"sync"
	"time"

	"strconv"
	"strings"
	"runtime"
	"encoding/json"

	"DaiZong/config"
	"DaiZong/elasticache"
	"DaiZong/queue"
	"DaiZong/util"
	"DaiZong/websocket"

	"github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/garyburd/redigo/redis"
)

type HandlerAckFunc func(msgid string, deviceid string, q *queue.Queue, message *sqs.Message) error

type HandlerAck interface {
	HandleAckMessage(msgid string, deviceid string, q *queue.Queue, message *sqs.Message) error
}

func (h HandlerAckFunc) HandleAckMessage(msgid string, deviceid string, q *queue.Queue, message *sqs.Message) error {
	return h(msgid, deviceid, q, message)
}

type HandlerRetryFunc func(msgid string, deviceid string, q *queue.Queue, m *sqs.Message, i int) error

type HandlerRetry interface {
	HandlerRetryMessage(msgid string, deviceid string, q *queue.Queue, m *sqs.Message, i int) error
}

func (retry HandlerRetryFunc) HandlerRetryMessage(msgid string, deviceid string, q *queue.Queue, m *sqs.Message, i int) error {
	return retry(msgid, deviceid, q, m, i)
}

var (
	redisconn *elasticache.AwsElastiCache
	log = logrus.New()
	visibility, _ = strconv.ParseInt(config.Conf.GetYamlValue("visibilitytimeout"), 10, 64)
	max, _ = strconv.ParseInt(config.Conf.GetYamlValue("maxnumber"), 10, 64)
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	websocket.NewHandler()
	redisconn = &elasticache.AwsElastiCache{}
	//go HandlerSQSMessage(config.Conf.GetYamlValue("high_priority_queue_name"), config.Conf.GetYamlValue("region"))
	go HandlerSQSMessage(config.Conf.GetYamlValue("common_queue_name"), config.Conf.GetYamlValue("region"))
//	go HandlerToAppMessage(config.Conf.GetYamlValue("region"))
	go HandlerRetryFailMessage()
}

func HandlerSQSMessage(out_queue_name string, region string) {
	defer func() {
		if err := recover(); err != nil {
			log.WithFields(logrus.Fields{"error":err}).Info("read sqs message to handler")
		}
	}()
	q, _ := queue.NewQueue(out_queue_name, region)

	for {
		mess, err := q.ReceiveMessage(visibility, max)
		if err != nil {
			log.WithFields(logrus.Fields{"error": err}).Info("read sqs message to handler ")
		}
		if mess != nil {
			for _, message := range mess.Messages {
				var user websocket.UserInfo
				json.Unmarshal([]byte(*message.Body), &user)
				if user.Types == websocket.REGISTERMESSAGE {
					go func() {
						reply, err := redisconn.Connection().Do("SET", user.Data["deviceid"], *message.Body)
						log.WithFields(logrus.Fields{"reply":reply, "redis_key":user.Data["deviceid"]}).Info("register info send to redis ")
						if err == nil {
							q.DeleteMessage(message.ReceiptHandle)
						}
					}()

				} else if user.Types == websocket.ACKMESSAGE {
					go func() {
						msgid, _ := user.Data["msgid"].(string)
						deviceid, _ := user.Data["deviceid"].(string)
						//_, err := redisconn.Connection().Do("SREM", msgid, deviceid)
						//if err != nil {
						//	log.WithFields(logrus.Fields{"err":err}).Info("handler ack info fail")
						//}
						q.DeleteMessage(message.ReceiptHandle)
						deleteAckmessage(msgid, deviceid, HandlerAckFunc(checkAckMessage))
					}()

				} else if user.Types == websocket.UNREGISTERMESSAGE {
					go func() {
						redisconn.Connection().Do("DEL", user.Data["deviceid"])
					}()

				} else if user.Types == websocket.ONOFFLINE {
					go func() {
						q.DeleteMessage(message.ReceiptHandle)

						m :=make(map[string]interface{})
						m["cid"] = user.Data["deviceid"]
						m["online"] = user.Status
						a := websocket.Argus{Types:"app-event",Data:m}
						b,_ :=json.Marshal(a)
						redisconn.Connection().Do("RPUSH", "Argus", string(b))
					}()

				} else if user.Types == websocket.CONTENT_INFO_DETAIL {
					go func() {
						HandlerInfoDetailMessage(q, user, message, region)
					}()

				}
			}
		}

		//time.Sleep(time.Microsecond * 16)
	}

}

func HandlerToAppMessage(region string) {
	in_queue_name := util.IpAddress()
	var wg sync.WaitGroup
	defer func() {
		if err := recover(); err != nil {
			wg.Done()
			log.WithFields(logrus.Fields{"error":err}).Info("get local machine message")
		}
	}()

	wg.Add(1)
	for {
		q, _ := queue.NewQueue(in_queue_name, region)
		mess, _ := q.ReceiveMessage(visibility, max)
		for _, message := range mess.Messages {

			var msg websocket.MessageInfo
			json.Unmarshal([]byte(*message.Body), &msg)
			var downdata websocket.DownStreamData
			json.Unmarshal(msg.Message, &downdata)
			ack := downdata.Ack
			deviceid := downdata.Deviceid
			msgid := downdata.Msgid
			flag := false
			for k, v := range websocket.Dp.Sessions {

				if v == deviceid {
					k.WriteRaw(&msg)
					flag = true
				}
			}
			if !flag {
				log.WithFields(logrus.Fields{"deviceid":deviceid}).Info("get local message to consumer,but this message deviceid not exist")
				q.DeleteMessage(message.ReceiptHandle)
				//redisconn.Connection().Do("SREM", msgid, deviceid)
				continue
			}
			if !ack {
				q.DeleteMessage(message.ReceiptHandle)
			}else {
				key := strings.Join([]string{msgid, deviceid}, "_")
				dt := strconv.FormatInt(time.Now().Unix(), 10)
				val := strings.Join([]string{dt, strconv.Itoa(0)}, "#")
				redisconn.Connection().Do("SET", key, val)
			}
		}
		time.Sleep(time.Microsecond * 22)
	}
	wg.Done()
}

//要单独控制ack的消息的visibility
func deleteAckmessage(msgid string, deviceid string, h HandlerAck) {
	q, _ := queue.NewQueue(util.IpAddress(), config.Conf.GetYamlValue("region"))
	for {
		mess, err := q.ReceiveMessage(visibility, max)
		if err != nil {
			log.WithFields(logrus.Fields{"error":err}).Info("delete message from ack")
			continue
		}
		run(msgid, deviceid, q, mess.Messages, h)
	}

}

func run(msgid string, deviceid string, q *queue.Queue, messages []*sqs.Message, h HandlerAck) {
	nums := len(messages)
	var wg sync.WaitGroup
	wg.Add(nums)
	for i, _ := range messages {
		go func(m *sqs.Message) {
			defer wg.Done()
			if err := handleackMessage(msgid, deviceid, q, m, h); err != nil {
				log.WithFields(logrus.Fields{"error":err}).Info("delete message from ack ")
			}
		}(messages[i])
	}
	wg.Wait()

}

func handleackMessage(msgid string, deviceid string, q *queue.Queue, m *sqs.Message, h HandlerAck) error {
	var err error
	err = h.HandleAckMessage(msgid, deviceid, q, m)
	if err != nil {
		return err
	}
	return nil
}

func checkAckMessage(msgid string, deviceid string, q *queue.Queue, message *sqs.Message) error {
	var user websocket.UserInfo
	json.Unmarshal([]byte(*message.Body), &user)

	if msgid == user.Data["msgid"] && deviceid == user.Data["deviceid"] {
		q.DeleteMessage(message.ReceiptHandle)
		key := strings.Join([]string{msgid, deviceid}, "_")
		_, err := redisconn.Connection().Do("DEL", key)
		if err != nil {
			return err
		}
	}
	return nil
}

func HandlerInfoDetailMessage(q *queue.Queue, user websocket.UserInfo, message *sqs.Message, region string) {
	var wg sync.WaitGroup
	defer func() {
		if err := recover(); err != nil {
			wg.Done()
		}
	}()
	wg.Add(1)
	msgid := util.MakeMsgID()
	var ack bool = user.Data["ack"].(bool)
	var ids []string
	bt, _ := json.Marshal(user.Data["deviceid"])
	json.Unmarshal(bt, &ids)

	info := user.Data["message"]
	if len(ids) > 0 {
		for _, id := range ids {
			//if ack {
			//	redisconn.Connection().Do("SADD", msgid, id)
			//}
			res, err := redis.Bytes(redisconn.Connection().Do("GET", id))
			if err != nil {
				log.WithFields(logrus.Fields{"error":err,"deviceid":id}).Info("get deviceid register info fail")
				continue
			}

			if len(res) <= 0 {
				log.WithFields(logrus.Fields{"result_len":len(res)}).Info("get deviceid register info")
				continue
			}
			var user websocket.UserInfo
			json.Unmarshal(res, &user)
			in, _ := queue.NewQueue(user.Servername, region)
			messagebyte, _ := json.Marshal(websocket.DownStreamData{Msgid: msgid, Deviceid:id, Ack: ack, Data:info})
			finalbyte, _ := json.Marshal(websocket.MessageInfo{Ws_type: websocket.TextMessage, Message: messagebyte})
			result := in.SendMessage(string(finalbyte[:len(finalbyte)]))
			if result == nil {
				q.DeleteMessage(message.ReceiptHandle)
			}
		}
	}
	wg.Done()
}

func HandlerRetryFailMessage() {
	retrycount, err := strconv.Atoi(config.Conf.GetYamlValue("retry_count"))
	if err != nil {
		log.WithFields(logrus.Fields{"error":err}).Info("read config attribute retry_count)")
	}
	retryinterval, err := strconv.Atoi(config.Conf.GetYamlValue("retry_interval"))
	if err != nil {
		log.WithFields(logrus.Fields{"error":err}).Info("read config attribute retry_interval)")
	}

	k := time.Now().Unix()
	loop:
	for {
		var key string
		replys, err := redis.Values(redisconn.Connection().Do("KEYS", "*_*"))
		if len(replys) <= 0 || err != nil {
			log.WithFields(logrus.Fields{"error":err}).Info("KEYS *_*")
		}
		redis.Scan(replys, &key)
		if key == "" {
			break loop
		}
		ks := strings.Split(key, "_")
		if len(ks) <= 0 {
			break loop
		}
		msgid := ks[0]
		deviceid := ks[1]
		value, _ := redis.String(redisconn.Connection().Do("GET", key))
		rs := strings.Split(value, "#")
		if len(rs) <= 0 {
			break loop
		}
		rc, _ := strconv.Atoi(rs[1])
		t, _ := strconv.ParseInt(rs[0], 10, 0)


		if rc <= retrycount  && k - t >= int64(retryinterval) {
			overnum := retrycount - rc

			send(msgid, deviceid, overnum)
		}
		time.Sleep(time.Microsecond * 5)
	}
}

func send(msgid string, deviceid string, i int) {
	q, _ := queue.NewQueue(util.IpAddress(), config.Conf.GetYamlValue("region"))
	for {
		mess, err := q.ReceiveMessage(visibility, max)
		if err != nil {
			log.WithFields(logrus.Fields{"error":err}).Info("get need to retry  message")
			continue
		}
		if len(mess.Messages) > 0 {
			RetrySend(msgid, deviceid, q, mess.Messages, i, HandlerRetryFunc(handlerRetry))
		}
	}
}

func RetrySend(msgid string, deviceid string, q *queue.Queue, messages []*sqs.Message, n int, h HandlerRetry) {
	nums := len(messages)
	var wg sync.WaitGroup
	wg.Add(nums)
	for i, _ := range messages {
		go func(m *sqs.Message) {
			defer wg.Done()
			if err := handlerRetryFailMessage(msgid, deviceid, q, m, n, h); err != nil {
				log.WithFields(logrus.Fields{"error":err}).Info("retry send message from no ack")
			}
		}(messages[i])
	}
	wg.Wait()
}

func handlerRetryFailMessage(msgid string, deviceid string, q *queue.Queue, m *sqs.Message, n int, h HandlerRetry) error {
	var err error
	err = h.HandlerRetryMessage(msgid, deviceid, q, m, n)
	if err != nil {
		log.WithFields(logrus.Fields{"error":err}).Info("")
		return err
	}
	if n == 1 {
		log.Info("===============delete sqs message================")
		_, err := q.DeleteMessage(m.ReceiptHandle)
		if err != nil {
			return err
		}
		log.Info("===============delete redis message================")
		key := strings.Join([]string{msgid, deviceid}, "_")
		_, err = redisconn.Connection().Do("DEL", key)
		return err
	}
	return nil

}

func handlerRetry(msgid string, deviceid string, q *queue.Queue, message *sqs.Message, n int) error {
	var msg websocket.MessageInfo
	json.Unmarshal([]byte(*message.Body), &msg)
	var downdata websocket.DownStreamData
	json.Unmarshal(msg.Message, &downdata)
        log.WithFields(logrus.Fields{"msgid":msgid," downdata.msgid":downdata.Msgid}).Info("test msgid retry")
	log.WithFields(logrus.Fields{"deviceid":deviceid," downdata.deviceid":downdata.Deviceid}).Info("test deviceid retry")

	if msgid == downdata.Msgid && deviceid == downdata.Deviceid {
		for k, v := range websocket.Dp.Sessions {
			log.Info("========")
			log.Info(k)
			log.Info(v)
			log.Info("========")
			if v == deviceid {
				k.WriteRaw(&msg)
				//回写次数
				retrycount, err := strconv.Atoi(config.Conf.GetYamlValue("retry_count"))
				if err != nil {
					log.WithFields(logrus.Fields{"error":err}).Info("read config attribute 'retry_count' ")
				}
				key := strings.Join([]string{msgid, deviceid}, "_")
				//value
				dt := strconv.FormatInt(time.Now().Unix(), 10)
				str1 := []string{dt, strconv.Itoa(retrycount - n + 1)}
				val := strings.Join(str1, "#")
				redisconn.Connection().Do("SET", key, val)
				log.Info("======================complete======================")
			}
		}
	}
	return nil
}
