package inmemory_test

import (
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/go-gnss/ntrip"
	"github.com/go-gnss/ntrip/internal/inmemory"
	"github.com/sirupsen/logrus"
)

type auth struct{}

func (_ *auth) Authorise(action inmemory.Action, mount string, username string, password string) (authorised bool, err error) {
	if username == "foo" {
		return false, fmt.Errorf("intentionally triggered auth error")
	}

	if username != "username" || password != "password" {
		return false, nil
	}

	return true, nil
}

// TODO: Actually write some tests for this, once I work out a direction for it
func _TestInMemoryService(t *testing.T) {
	caster := ntrip.NewCaster(":2101", inmemory.NewSourceService(&auth{}), logrus.StandardLogger())

	go func() {
		r, w := io.Pipe()
		for {
			req, _ := ntrip.NewServerRequest("http://localhost:2101/TEST00AUS0", r)
			req.SetBasicAuth("username", "password")
			resp, err := http.DefaultClient.Do(req)
			if err == nil && resp.StatusCode == 200 {
				break
			}
			fmt.Println(resp, err)
			time.Sleep(100 * time.Millisecond)
		}

		for {
			fmt.Fprintf(w, "%s\n", time.Now())
			time.Sleep(100 * time.Millisecond)
		}
	}()

	caster.ListenAndServe()
}
