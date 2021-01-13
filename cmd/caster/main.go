package main

import (
	"github.com/go-gnss/ntrip"
	"github.com/go-gnss/ntrip/internal/inmemory"
	"github.com/sirupsen/logrus"
)

func main() {
	logger := logrus.StandardLogger()
	auth := &Auth{Users: make(map[string]string)}
	st := ntrip.Sourcetable{}
	svc := inmemory.NewSourceService(st, auth)

	conf, err := Config(logger, auth, svc)
	if err != nil {
		logger.WithError(err).Fatal("failed to read config")
	}

	caster := ntrip.NewCaster(
		conf.GetString(ConfKeyServerAddress), svc, logger)

	logger.Infof("starting caster on address: %s", caster.Addr)
	logger.Fatalf("caster stopped with reason: %s", caster.ListenAndServe())
}
