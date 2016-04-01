package util

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"math/rand"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	log         = logrus.New()
	hostname    string
	getHostname sync.Once
)

func IpAddress() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.WithFields(logrus.Fields{"err": err}).Info("get ipaddress fail")
	}
	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return strings.Replace(ipnet.IP.String(), ".", "_", -1)
			}
		}
	}
	return "127_0_0_1"
}

func MakeMsgID() string {
	getHostname.Do(func() {
		var err error
		if hostname, err = os.Hostname(); err != nil {
			hostname = "localhost"
		}
	})
	now := time.Now()
	return fmt.Sprintf("%d&%d&%d@%s", now.Unix(), now.UnixNano(), rand.Int63(), hostname)
}