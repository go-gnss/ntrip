// Example of how to implement Client and Server to relay streams
package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/go-gnss/ntrip"
)

var (
	reader, writer = io.Pipe()
)

func main() {
	source := flag.String("source", "", "Source NTRIP caster URL to stream from")
	sourceUsername := flag.String("suser", "", "Username for accessing the Source NTRIP caster")
	sourcePassword := flag.String("spass", "", "Password for accessing the Source NTRIP caster")
	destination := flag.String("dest", "", "NTRIP caster URL to stream from")
	destUsername := flag.String("duser", "", "Username for accessing the Destination NTRIP caster")
	destPassword := flag.String("dpass", "", "Password for accessing the Destination NTRIP caster")
	timeout := flag.Duration("timeout", 2, "NTRIP reconnect timeout")
	flag.Parse()

	go serve(*destination, *destUsername, *destPassword, *timeout)

	// Write response body to PipeWriter
	client, _ := ntrip.NewClientRequest(*source)
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
			writer.Write(data[:br])
		}

		fmt.Println("client connection died", err)
	}
}

// Serve whatever is written to the PipeWriter
func serve(url, username, password string, timeout time.Duration) {
	for ; ; time.Sleep(time.Second * timeout) {
		reader, writer = io.Pipe()
		req, _ := ntrip.NewServerRequest(url, reader)
		req.SetBasicAuth(username, password)
		resp, err := http.DefaultClient.Do(req)
		if err != nil || resp.StatusCode != 200 {
			fmt.Println("server failed to connect", resp, err)
			continue
		}
		fmt.Println("server connected")
		io.ReadAll(resp.Body)
		fmt.Println("server connection died")
	}
}
