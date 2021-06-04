package ntrip_test

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/go-gnss/ntrip"
)

func ExampleNewClientRequest_sourcetable() {
	req, _ := ntrip.NewClientRequest("https://ntrip.data.gnss.ga.gov.au")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("error making NTRIP request: %s", err)
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("received non-200 response code: %d", resp.StatusCode)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("error reading from response body")
	}

	sourcetable, _ := ntrip.ParseSourcetable(string(data))
	fmt.Printf("caster has %d mountpoints available", len(sourcetable.Mounts))
}

func ExampleNewClientRequest() {
	req, _ := ntrip.NewClientRequest("http://hostname:2101/mountpoint")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("error making NTRIP request: %s", err)
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("received non-200 response code: %d", resp.StatusCode)
	}

	// Read from resp.Body until EOF
}

func ExampleNewServerRequest() {
	r, w := io.Pipe()

	req, _ := ntrip.NewServerRequest("http://hostname:2101/mountpoint", r)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("error making NTRIP request: %s", err)
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("received non-200 response code: %d", resp.StatusCode)
	}

	w.Write([]byte("write data to the NTRIP caster"))
	w.Close()
}
