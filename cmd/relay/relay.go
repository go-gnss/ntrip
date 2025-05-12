// Example of how to implement Client and Server to relay streams
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
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

	// Create a properly configured HTTP client
	client := ntrip.DefaultHTTPClient()

	for ; ; time.Sleep(time.Second * *timeout) {
		// Create a context with timeout for each connection attempt
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

		// Create a new request with context
		req, _ := ntrip.NewClientRequestWithContext(ctx, *source)
		req.SetBasicAuth(*sourceUsername, *sourcePassword)

		// Make the request
		resp, err := client.Do(req)
		if err != nil || resp.StatusCode != 200 {
			fmt.Println("client failed to connect", resp, err)
			cancel() // Cancel the context
			continue
		}

		fmt.Println("client connected")

		// Create a buffer pool for efficient memory reuse
		bufPool := make([]byte, 4096)

		// Copy data from response to writer
		for {
			br, err := resp.Body.Read(bufPool)
			if err != nil {
				break
			}
			if _, err := writer.Write(bufPool[:br]); err != nil {
				break
			}
		}

		fmt.Println("client connection died", err)
		cancel()          // Cancel the context
		resp.Body.Close() // Ensure response body is closed
	}
}

// Serve whatever is written to the PipeWriter
func serve(url, username, password string, timeout time.Duration) {
	// Create a properly configured HTTP client
	client := ntrip.DefaultHTTPClient()

	for ; ; time.Sleep(time.Second * timeout) {
		// Create a context with timeout for each connection attempt
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

		reader, writer = io.Pipe()

		// Create a new request with context
		req, _ := ntrip.NewServerRequestWithContext(ctx, url, reader)
		req.SetBasicAuth(username, password)

		// Make the request
		resp, err := client.Do(req)
		if err != nil || resp.StatusCode != 200 {
			fmt.Println("server failed to connect", resp, err)
			cancel() // Cancel the context
			continue
		}

		fmt.Println("server connected")

		// Read response body until EOF
		io.ReadAll(resp.Body)

		fmt.Println("server connection died")
		cancel()          // Cancel the context
		resp.Body.Close() // Ensure response body is closed
	}
}
