package ntrip_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/go-gnss/ntrip"
)

func ExampleNewClientRequest_sourcetable() {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Use context-aware request
	req, _ := ntrip.NewClientRequestWithContext(ctx, "https://ntrip.data.gnss.ga.gov.au")

	// Use properly configured client
	client := ntrip.DefaultHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("error making NTRIP request: %s", err)
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("received non-200 response code: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("error reading from response body")
	}

	sourcetable, _ := ntrip.ParseSourcetable(string(data))
	fmt.Printf("caster has %d mountpoints available", len(sourcetable.Mounts))
}

func ExampleNewClientRequest() {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Use context-aware request
	req, _ := ntrip.NewClientRequestWithContext(ctx, "http://hostname:2101/mountpoint")

	// Use properly configured client
	client := ntrip.DefaultHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("error making NTRIP request: %s", err)
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("received non-200 response code: %d", resp.StatusCode)
	}

	// Read from resp.Body until EOF
}

func ExampleNewServerRequest() {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	r, w := io.Pipe()

	// Use context-aware request
	req, _ := ntrip.NewServerRequestWithContext(ctx, "http://hostname:2101/mountpoint", r)

	// Use properly configured client
	client := ntrip.DefaultHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("error making NTRIP request: %s", err)
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("received non-200 response code: %d", resp.StatusCode)
	}

	w.Write([]byte("write data to the NTRIP caster"))
	w.Close()
}
