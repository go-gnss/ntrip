package main

import (
	"fmt"

	"github.com/fsnotify/fsnotify"
	"github.com/go-gnss/ntrip"
	"github.com/go-gnss/ntrip/internal/inmemory"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const (
	ConfKeyServerAddress string = "server.address"
	ConfKeySourcetable   string = "sourcetable"
	ConfKeyUsers         string = "users"
	ConfKeyLogLevel      string = "logging.debug"
)

func Config(logger *logrus.Logger, auth *Auth, svc *inmemory.SourceService) (*viper.Viper, error) {
	conf := viper.New()
	// TODO: Pass in as flag or get from env
	conf.SetConfigName("config")
	conf.SetConfigType("yaml")
	conf.AddConfigPath(".")

	if err := conf.ReadInConfig(); err != nil {
		return nil, err
	}

	if err := updateFromConfig(conf, auth, svc, logger); err != nil {
		return conf, err
	}

	conf.OnConfigChange(func(event fsnotify.Event) {
		if err := updateFromConfig(conf, auth, svc, logger); err != nil {
			logger.WithError(err).Error("error updating config")
		}
	})
	conf.WatchConfig()

	return conf, nil
}

func updateFromConfig(conf *viper.Viper, auth *Auth, svc *inmemory.SourceService, logger *logrus.Logger) error {
	// TODO: Probably don't actually want to do this, should probably actually load mounts as STR sourcetable strings for compactness
	var st ntrip.Sourcetable
	err := conf.UnmarshalKey(ConfKeySourcetable, &st)
	if err != nil {
		return fmt.Errorf("error reading sourcetable: %s", err)
	}
	svc.UpdateSourcetable(st)

	if conf.GetBool(ConfKeyLogLevel) {
		logger.SetLevel(logrus.DebugLevel)
	}

	// TODO: Add warnings for important missing config items, like users
	// TODO: Validate that passwords are valid bcrypt strings?
	auth.Users = conf.GetStringMapString(ConfKeyUsers)

	return nil
}
