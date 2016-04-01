package main

import (
	"os"
	"flag"
	"net/http"
	"runtime"

	"DaiZong/config"
	"DaiZong/websocket"
	_ "DaiZong/server"

	"github.com/Sirupsen/logrus"


)

var (
	address = flag.String("address", config.Conf.GetYamlValue("daizong_server_address"), "http server address")
	Log     = logrus.New()
)

func init() {
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.WarnLevel)
}

func main() {
	flag.Parse()
	count := runtime.GOMAXPROCS(runtime.NumCPU())
	Log.WithFields(logrus.Fields{"numcpu": count}).Info("record server cpu")
	handler := websocket.NewHandler()
	http.HandleFunc("/login", handler.HandleRequest)
	if err := http.ListenAndServe(*address, nil); err != nil {
		Log.WithFields(logrus.Fields{"listenAndServer": *address}).Info("servere listen fail")
	}
}
