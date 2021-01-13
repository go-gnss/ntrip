package main

import (
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/go-gnss/ntrip"
	"github.com/sirupsen/logrus"
)

// Spins up 60 NTRIP Servers and 10 clients per server
func Test(t *testing.T) {
	logger := logrus.StandardLogger()
	logger.SetLevel(logrus.DebugLevel)
	for i := range make([]byte, 10) {
		for j := range make([]byte, 6) {
			mount := fmt.Sprintf("TEST0%dAUS%d", i, j)
			go Serve(logger, mount)
			for k := range make([]byte, 10) {
				go Client(logger, mount, k)
			}
		}
	}

	main()
}

func Serve(logger *logrus.Logger, mount string) {
	for ; ; time.Sleep(1 * time.Second) {
		r, w := io.Pipe()
		req, _ := ntrip.NewServerRequestV2("http://localhost:2101/"+mount, r)
		req.SetBasicAuth("user1", "password")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			logger.WithError(err).Error("server failed to connect")
			continue
		}

		if resp.StatusCode != 200 {
			logger.Errorf("server received incorrect status code %d", resp.StatusCode)
			continue
		}

		logger.Debugf("server %s connected", mount)

		for ; ; time.Sleep(100 * time.Millisecond) {
			_, err := w.Write([]byte(time.Now().Format(time.RFC3339Nano) + "\r\n"))
			if err != nil {
				logger.WithError(err).Error("server failed to write to caster")
				r.Close()
				break
			}
		}
	}
}

func Client(logger *logrus.Logger, mount string, id int) {
	for ; ; time.Sleep(1 * time.Second) {
		req, _ := ntrip.NewClientRequestV2("http://localhost:2101/" + mount)
		req.SetBasicAuth("user2", "password")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			logger.WithError(err).Errorf("client %s:%d failed to connect", mount, id)
			continue
		}

		if resp.StatusCode != 200 {
			logger.Errorf("client %s:%d received incorrect status code %d", mount, id, resp.StatusCode)
		}

		logger.Debugf("client %s:%d connected", mount, id)

		for {
			buf := make([]byte, 1024)
			br, err := resp.Body.Read(buf)
			if err != nil {
				logger.WithError(err).Errorf("client %s:%d failed to read from caster", mount, id)
				resp.Body.Close()
				break
			}

			t, err := time.Parse(time.RFC3339Nano, string(buf[:br-2]))
			if err != nil {
				logger.WithError(err).Errorf("client %s:%d read invalid string from caster: %s", mount, id, buf[:br])
				continue
			}

			fmt.Printf("%s,%s:%d,%s\n", time.Now().Format(time.RFC3339Nano), mount, id, time.Now().Sub(t))
		}
	}
}
