// Example of how to implement Client and Server to relay streams
package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/go-gnss/ntrip"
)

func main() {
	source := flag.String("source", "", "Source NTRIP caster mountpoint to stream from")
	sourceUsername := flag.String("suser", "", "Username for accessing the Source NTRIP caster")
	sourcePassword := flag.String("spass", "", "Password for accessing the Source NTRIP caster")
	destination := flag.String("destination", "", "NTRIP caster mountpoint to stream from")
	destUsername := flag.String("duser", "", "Username for accessing the Destination NTRIP caster")
	destPassword := flag.String("dpass", "", "Password for accessing the Destination NTRIP caster")
	timeout := flag.Duration("timeout", 2, "NTRIP reconnect timeout")
	flag.Parse()

	r, w := io.Pipe()
	// Serve whatever is written to the PipeWriter
	go serve(*destination, *destUsername, *destPassword, *timeout, r)

	// Write response body to PipeWriter
	client, _ := ntrip.NewClientRequestV2(*source)
	client.SetBasicAuth(*sourceUsername, *sourcePassword)
	for ; ; time.Sleep(time.Second * *timeout) {
		resp, err := http.DefaultClient.Do(client)
		if err != nil || resp.StatusCode != 200 {
			fmt.Println("client failed to connect", resp, err)
			continue
		}

		fmt.Println("client connected")
		data := make([]byte, 4096)
		br, err := resp.Body.Read(data)
		for ; err == nil; br, err = resp.Body.Read(data) {
			w.Write(data[:br])
		}

		fmt.Println("client connection died", err)
	}
}

func serve(url, username, password string, timeout time.Duration, body io.ReadCloser) {
	req, _ := ntrip.NewServerRequestV2(url, body)
	req.SetBasicAuth(username, password)
	go func() {
		for ; ; time.Sleep(time.Second * timeout) {
			resp, err := http.DefaultClient.Do(req)
			if err != nil || resp.StatusCode != 200 {
				fmt.Println("server failed to connect", resp, err)
				continue
			}
			fmt.Println("server connected")
			ioutil.ReadAll(resp.Body)
			fmt.Println("server connection died")
		}
	}()
}
