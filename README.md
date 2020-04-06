NTRIP Caster / Client / Server implementation in Go

[![Go Report Card](https://goreportcard.com/badge/github.com/go-gnss/ntrip)](https://goreportcard.com/report/github.com/go-gnss/ntrip)

### Run a Caster 
Application in `cmd/ntripcaster/` configurable with `cmd/ntripcaster/caster.json`.

```
# Generate self signed certs for testing
openssl genrsa -out key.pem 2048
openssl req -new -x509 -sha256 -key server.key -out cert.pem -days 3650

ntripcaster &
curl https://localhost:2102/mount -d "TEST" -i -k -u username:password &
curl http://localhost:2101/mount -i
```
