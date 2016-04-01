package elasticache

import (
	"flag"
	log "github.com/Sirupsen/logrus"
	"github.com/garyburd/redigo/redis"
	"time"
	"DaiZong/config"
)

var (
	redisAdress = flag.String("awsElastiCacheAdrress", config.Conf.GetYamlValue("elasticache_endpoint"), "ElastiCache Server Address")

	//maxConnections = flag.String("maxconn", 20, "max connections to elasticache")
)

type AwsElasticCacheConn interface {
	Do(cmd string, args ...interface{}) (interface{}, error)

	Send(cmd string, args ...interface{}) error

	Close()
}

type AwsElastiCache struct {
	conn redis.Conn
}

func (c *AwsElastiCache) Connection() redis.Conn {

	flag.Parse()
	cn, err := redis.DialTimeout("tcp",*redisAdress, 0, 1*time.Second, 1*time.Second)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Info("connection elastiCache fail !")
	}
	c.conn = cn
	return c.conn

}

func (c *AwsElastiCache) Close() {
	c.conn.Close()
}

func (c *AwsElastiCache) Do(cmd string, args ...interface{}) (interface{}, error) {

	con := c.conn
	defer con.Close()

	return con.Do(cmd, args...)

}

func (c *AwsElastiCache) Send(cmd string, args ...interface{}) error {
	con := c.conn
	defer con.Close()

	return con.Send(cmd, args...)

}

func (c *AwsElastiCache) Subscribe(channel ...interface{}) error {
	con := c.conn
	defer con.Close()
	con.Send("SUBSCRIBE", channel...)
	return con.Flush()
}

func (c *AwsElastiCache) PSubscribe(channel ...interface{}) error {
	con := c.conn
	defer con.Close()
	con.Send("PSUBSCRIBE", channel...)
	return con.Flush()
}

func (c *AwsElastiCache) Publish(channelName string, message string) error {
	con := c.conn
	defer con.Close()
	con.Send("PUBLISH", channelName, message)
	return con.Flush()
}

func (c *AwsElastiCache) Unsubscribe(channel ...interface{}) error {
	con := c.conn
	defer con.Close()
	con.Send("UNSUBSCRIBE", channel...)
	return con.Flush()
}

func (c *AwsElastiCache) PUnsubscribe(channel ...interface{}) error {
	con := c.conn
	defer con.Close()
	con.Send("PUNSUBSCRIBE", channel...)
	return con.Flush()
}
