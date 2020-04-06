// Example of how to implement Client and Server to relay streams
package main

import (
	"flag"
	"fmt"
	"github.com/go-gnss/ntrip"
	"io/ioutil"
	"time"
)

func main() {
	source := flag.String("source", "", "NTRIP caster mountpoint to stream from")
	destination := flag.String("destination", "", "NTRIP caster mountpoint to stream from")
	timeout := flag.Duration("timeout", 2, "NTRIP reconnect timeout")
	flag.Parse()

	server, _ := ntrip.NewServer(*destination)
	go func() {
		for ; ; time.Sleep(time.Second * *timeout) {
			resp, err := server.Connect()
			if err != nil || resp.StatusCode != 200 {
				fmt.Println("server failed to connect", resp, err)
				continue
			}
			fmt.Println("server connected")
			ioutil.ReadAll(resp.Body)
			fmt.Println("server connection died")
		}
	}()

	client, _ := ntrip.NewClient(*source)
	for ; ; time.Sleep(time.Second * *timeout) {
		resp, err := client.Connect()
		if err != nil || resp.StatusCode != 200 {
			fmt.Println("client failed to connect", resp, err)
			continue
		}

		fmt.Println("client connected")
		data := make([]byte, 4096)
		br, err := resp.Body.Read(data)
		for ; err == nil; br, err = resp.Body.Read(data) {
			server.Write(data[:br])
		}

		fmt.Println("client connection died", err)
	}
}
