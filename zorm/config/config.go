package config

import (
	"flag"
	"github.com/BurntSushi/toml"
	zormlog "github.com/caixr9527/zorm/log"
	"os"
)

var Conf = &ZormConfig{
	logger: zormlog.Default(),
}

type ZormConfig struct {
	logger   *zormlog.Logger
	Log      map[string]any
	Pool     map[string]any
	Template map[string]any
}

func init() {
	loadToml()
}

func loadToml() {
	configFile := flag.String("conf", "conf/app.toml", "app config file")
	flag.Parse()
	if _, err := os.Stat(*configFile); err != nil {
		Conf.logger.Error("conf/app.toml file not load, because not exist")
		return
	}
	_, err := toml.DecodeFile(*configFile, Conf)
	if err != nil {
		Conf.logger.Error("conf/app.toml decode fail check format")
		return
	}
}
