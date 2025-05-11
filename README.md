# NTRIP Caster / Client / Server implementation in Go

[![Go Report Card](https://goreportcard.com/badge/github.com/go-gnss/ntrip)](https://goreportcard.com/report/github.com/go-gnss/ntrip)

## Overview

This package provides a complete implementation of the NTRIP (Networked Transport of RTCM via Internet Protocol) protocol in Go. It supports both NTRIP version 1 and 2, as well as additional features like RTSP/RTP support, sourcetable filtering, advanced authentication mechanisms, and NTRIP v1 SOURCE request handling.

## Features

### Core NTRIP Features
- NTRIP v1 and v2 client and server support
- Sourcetable parsing and generation
- Chunked transfer encoding for v2
- Basic authentication

### Advanced Features
- **RTSP/RTP Support**: Stream GNSS data over RTSP/RTP protocol
- **Sourcetable Filtering**: Filter sourcetable entries based on various criteria
- **Advanced Authentication**: Support for Basic, Digest, and Bearer authentication methods
- **NTRIP v1 SOURCE Request Handling**: Support for NTRIP v1 SOURCE requests

## Usage Examples

### Client Examples
Examples of NTRIP client implementations in [client_test.go](/client_test.go).

### Caster Examples
An example of setting up a Caster can be found in [internal/inmemory](/internal/inmemory/service_test.go).

### Complete Server Example
A complete server example with all features can be found in [cmd/ntrip-server](/cmd/ntrip-server/main.go).

## Sourcetable Filtering

The sourcetable filtering feature allows clients to request a filtered view of the sourcetable based on various criteria. The filter query can be specified in the URL query string.

Example filter queries:
```
?STR;;;;;;DEU                 # Filter streams in Germany
?&Bitrate>5000               # Filter streams with bitrate > 5000
?&NavSystem~GAL              # Filter streams that include Galileo
?&CountryCode=USA&NMEA=true  # Filter streams in USA with NMEA support
```

## Authentication

The package supports multiple authentication methods:

- **Basic Authentication**: Standard HTTP Basic authentication
- **Digest Authentication**: More secure authentication method that doesn't transmit passwords in clear text
- **Bearer Authentication**: Token-based authentication

Authentication can be configured per mount point, allowing different authentication methods for different streams.

## RTSP/RTP Support

The RTSP/RTP support allows streaming GNSS data over the RTSP protocol, which is more suitable for some network configurations and can provide better performance in certain scenarios.

The RTSP server supports the standard RTSP methods (OPTIONS, DESCRIBE, SETUP, PLAY, PAUSE, TEARDOWN) and generates appropriate SDP descriptions for GNSS data streams.

## NTRIP v1 SOURCE Request Handling

The package includes support for NTRIP v1 SOURCE requests, which allows NTRIP v1 servers to connect to the caster and provide data streams. This is implemented as a separate server that listens on a dedicated port.
