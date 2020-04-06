package main

import (
	"flag"
	"github.com/go-gnss/ntrip/caster"
	"github.com/go-gnss/ntrip/caster/authorizers"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"time"
)

func main() {
	ntripcaster := caster.Caster{
		Mounts:  make(map[string]*caster.Mountpoint),
		Timeout: 5 * time.Second,
	} // TODO: Hide behind NewCaster which can include a DefaultAuthenticator
	log.SetFormatter(&log.JSONFormatter{})

	configFile := flag.String("config", "cmd/ntripcaster/caster.json", "Path to config file")
	flag.Parse()

	conf := Config{}
	viper.SetConfigFile(*configFile)
	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}

	err = viper.Unmarshal(&conf)
	if err != nil {
		panic(err)
	}

	ntripcaster.Authorizer = authorizers.NewCognitoAuthorizer(conf.Cognito.UserPoolID, conf.Cognito.ClientID)

	panic(ntripcaster.ListenHTTP(conf.HTTP.Port))
}
