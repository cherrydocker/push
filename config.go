package config

import (
	"io/ioutil"
	"path/filepath"
	"sync"

	"github.com/Sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

const (
	DEFAULT_CONFIG_FILE_NAME = "config/env.yml"
)

var (
	config map[string]interface{}
	Conf   *CacheConfig
	once   sync.Once
	log    = logrus.New()
)

type CacheConfig struct {
}

func init() {
	NewCacheConfig()
	Conf.ParseYamlConfig("")
}

func (conf *CacheConfig) ParseYamlConfig(file string) map[string]interface{} {
	var filename string
	var err error
	if len(file) == 0 {
		filename, err = filepath.Abs(DEFAULT_CONFIG_FILE_NAME)
	} else {
		filename, err = filepath.Abs(file)
	}
	if err != nil {
		log.WithFields(logrus.Fields{"err": err}).Info("get config file absolute path  fail")
	}
	yml_file, err := ioutil.ReadFile(filename)
	if err != nil {
		log.WithFields(logrus.Fields{"config_file_name": filename, "err": err}).Info("read config file fail")
	}
	config = make(map[string]interface{})
	err = yaml.Unmarshal(yml_file, &config)
	if err != nil {
		log.WithFields(logrus.Fields{"err": err}).Info("read config file fail")
	}
	return config
}

func (conf *CacheConfig) GetYamlValue(key string) string {
	//todo  need to accord to different type conversion
	for k, v := range config {
		if k == key {
			return v.(string)
		}
	}
	return ""
}

func NewCacheConfig() *CacheConfig {
	once.Do(func() {
		Conf = &CacheConfig{}
	})
	return Conf
}
