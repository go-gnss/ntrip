package ntrip

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// Sourcetable for NTRIP Casters, returned at / as a way for users to discover available mounts
type Sourcetable struct {
	Casters  []CasterEntry
	Networks []NetworkEntry
	Mounts   []StreamEntry
}

func (st Sourcetable) String() string {

	stLength := (len(st.Casters) + len(st.Networks) + len(st.Mounts) + 1)
	stStrs := make([]string, 0, stLength)

	for _, cas := range st.Casters {
		stStrs = append(stStrs, cas.String())
	}

	for _, net := range st.Networks {
		stStrs = append(stStrs, net.String())
	}

	for _, str := range st.Mounts {
		stStrs = append(stStrs, str.String())
	}

	stStrs = append(stStrs, "ENDSOURCETABLE\r\n")
	return strings.Join(stStrs, "\r\n")
}

// CasterEntry for an NTRIP Sourcetable
type CasterEntry struct {
	Host                string
	Port                int
	Identifier          string
	Operator            string
	NMEA                bool
	Country             string
	Latitude            float32
	Longitude           float32
	FallbackHostAddress string
	FallbackHostPort    int
	Misc                string
}

func (c CasterEntry) String() string {
	nmea := "0"
	if c.NMEA {
		nmea = "1"
	}

	port := strconv.FormatInt(int64(c.Port), 10)
	fallbackPort := strconv.FormatInt(int64(c.FallbackHostPort), 10)

	lat := strconv.FormatFloat(float64(c.Latitude), 'f', 4, 32)
	lng := strconv.FormatFloat(float64(c.Longitude), 'f', 4, 32)

	return strings.Join([]string{
		"CAS", c.Host, port, c.Identifier, c.Operator, nmea, c.Country, lat, lng,
		c.FallbackHostAddress, fallbackPort, c.Misc,
	}, ";")
}

// NetworkEntry for an NTRIP Sourcetable
type NetworkEntry struct {
	Identifier string
	Operator   string
	// TODO: Authentication type - spec says: B, D, N or a comma separated list of these
	Authentication string
	Fee            bool
	NetworkInfoURL string
	StreamInfoURL  string
	// RegistrationAddress is either a URL or Email address
	RegistrationAddress string
	Misc                string
}

func (n NetworkEntry) String() string {
	fee := "N"
	if n.Fee {
		fee = "Y"
	}

	return strings.Join([]string{"NET",
		n.Identifier, n.Operator, n.Authentication, fee, n.NetworkInfoURL, n.StreamInfoURL,
		n.RegistrationAddress, n.Misc}, ";")
}

// StreamEntry for an NTRIP Sourcetable
type StreamEntry struct {
	Name          string
	Identifier    string
	Format        string
	FormatDetails string
	Carrier       string
	NavSystem     string
	Network       string
	CountryCode   string
	Latitude      float32
	Longitude     float32
	NMEA          bool
	Solution      bool
	Generator     string
	Compression   string
	// TODO: Authentication type
	Authentication string
	Fee            bool
	Bitrate        int
	Misc           string
}

// String representation of Mount in NTRIP Sourcetable entry format
func (m StreamEntry) String() string {
	nmea := "0"
	if m.NMEA {
		nmea = "1"
	}

	solution := "0"
	if m.Solution {
		solution = "1"
	}

	fee := "N"
	if m.Fee {
		fee = "Y"
	}

	bitrate := strconv.FormatInt(int64(m.Bitrate), 10)

	lat := strconv.FormatFloat(float64(m.Latitude), 'f', 4, 32)
	lng := strconv.FormatFloat(float64(m.Longitude), 'f', 4, 32)

	// Returning joined strings significantly reduced allocs when benchmarking. The old code is
	// commented out below for further analysis. There is a benchmark test that can be used
	// to compare these results:
	// go test ./... -run none -bench=. -benchmem -benchtime 3s
	// Make sure your computer is somewhat idle before running benchmarks.
	return strings.Join([]string{
		"STR", m.Name, m.Identifier, m.Format, m.FormatDetails, m.Carrier, m.NavSystem,
		m.Network, m.CountryCode, lat, lng,
		nmea, solution, m.Generator, m.Compression, m.Authentication, fee, bitrate, m.Misc,
	}, ";")

	// return fmt.Sprintf("STR;%s;%s;%s;%s;%s;%s;%s;%s;%.4f;%.4f;%s;%s;%s;%s;%s;%s;%d;%s",
	// m.Name, m.Identifier, m.Format, m.FormatDetails, m.Carrier, m.NavSystem, m.Network,
	// m.CountryCode, m.Latitude, m.Longitude, nmea, solution, m.Generator, m.Compression,
	// m.Authentication, fee, m.Bitrate, m.Misc)
}

// GetSourcetable fetches a source table from a specific caster.
//
// The funciton returns a list of errors which can be treated as warnings.
// These warnings indicate that the caster is returning an improper rtcm3 format.
func GetSourcetable(ctx context.Context, url string) (Sourcetable, []error, error) {
	warnings := []error{}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Sourcetable{}, warnings, errors.Wrap(err, "building request")
	}

	req.Header.Set("Ntrip-Version", "Ntrip/2.0")
	req.Header.Set("User-Agent", "ntrip-mqtt-gateway")

	client := &http.Client{}

	res, err := client.Do(req)
	if err != nil {
		return Sourcetable{}, warnings, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return Sourcetable{}, warnings, err
	}

	if res.StatusCode != 200 {
		return Sourcetable{}, warnings, fmt.Errorf("received a non 200 status code")
	}

	// Swollowing the errors here is okay because the errors are more like warnings.
	// All rows that could be parsed will be present in the source table.
	table, warnings := ParseSourcetable(string(body[:]))
	return table, warnings, nil
}

// ParseSourcetable parses a sourcetable from an ioreader into a ntrip style source table.
func ParseSourcetable(str string) (Sourcetable, []error) {
	table := Sourcetable{}
	var allErrors []error

	lines := strings.Split(str, "\n")

	for lineNo, rawLine := range lines {
		line := strings.TrimSpace(rawLine)

		if line == "" {
			continue
		}

		if line == "ENDSOURCETABLE" {
			break
		}

		switch line[:3] {
		case "CAS":
			caster, errs := ParseCasterEntry(line)
			if len(errs) != 0 {
				for _, err := range errs {
					allErrors = append(allErrors, errors.Wrapf(err, "parsing line %v", lineNo))
				}
			}
			table.Casters = append(table.Casters, caster)
		case "NET":
			net, errs := ParseNetworkEntry(line)
			if len(errs) != 0 {
				for _, err := range errs {
					allErrors = append(allErrors, errors.Wrapf(err, "parsing line %v", lineNo))
				}
			}
			table.Networks = append(table.Networks, net)
		case "STR":
			mount, errs := ParseStreamEntry(line)
			if len(errs) != 0 {
				for _, err := range errs {
					allErrors = append(allErrors, errors.Wrapf(err, "parsing line %v", lineNo))
				}
			}
			table.Mounts = append(table.Mounts, mount)
		}

	}

	return table, allErrors
}

// ParseCasterEntry parses a single caster from a string.
func ParseCasterEntry(casterString string) (CasterEntry, []error) {
	parts := strings.Split(casterString, ";")

	p := &parser{parts, []error{}}

	return CasterEntry{
		Host:                p.parseString(1, "host"),
		Port:                p.parseInt(2, "port"),
		Identifier:          p.parseString(3, "identifier"),
		Operator:            p.parseString(4, "operator"),
		NMEA:                p.parseBool(5, "0", "nmea"),
		Country:             p.parseString(6, "country"),
		Latitude:            p.parseFloat32(7, "latitude"),
		Longitude:           p.parseFloat32(8, "longitude"),
		FallbackHostAddress: p.parseString(9, "fallback host address"),
		FallbackHostPort:    p.parseInt(10, "fallback host port"),
		Misc:                p.parseString(11, "misc"),
	}, p.errors

}

// ParseNetworkEntry parses a single network entry from a string.
func ParseNetworkEntry(netString string) (NetworkEntry, []error) {
	parts := strings.Split(netString, ";")

	p := &parser{parts, []error{}}

	return NetworkEntry{
		Identifier:          p.parseString(1, "identifier"),
		Operator:            p.parseString(2, "operator"),
		Authentication:      p.parseString(3, "authentication"),
		Fee:                 p.parseBool(4, "N", "fee"),
		NetworkInfoURL:      p.parseString(5, "network info url"),
		StreamInfoURL:       p.parseString(6, "stream info url"),
		RegistrationAddress: p.parseString(7, "registration address"),
		Misc:                p.parseString(8, "misc"),
	}, p.errors

}

// ParseStreamEntry parses a single mount entry.
func ParseStreamEntry(streamString string) (StreamEntry, []error) {
	parts := strings.Split(streamString, ";")

	p := &parser{parts, []error{}}

	streamEntry := StreamEntry{
		Name:          p.parseString(1, "name"),
		Identifier:    p.parseString(2, "identifier"),
		Format:        p.parseString(3, "format"),
		FormatDetails: p.parseString(4, "format details"),
		Carrier:       p.parseString(5, "carrier"),
		NavSystem:     p.parseString(6, "nav system"),
		Network:       p.parseString(7, "network"),
		CountryCode:   p.parseString(8, "country code"),
		Latitude:      p.parseFloat32(9, "latitude"),
		Longitude:     p.parseFloat32(10, "logitude"),
		NMEA:          p.parseBool(11, "0", "nmea"),
		Solution:      p.parseBool(12, "0", "solution"),
		Generator:     p.parseString(13, "generator"),
		Compression:   p.parseString(14, "compression"),
		// TODO: Authentication type
		Authentication: p.parseString(15, "authentication"),
		Fee:            p.parseBool(16, "N", "fee"),
		Bitrate:        p.parseInt(17, "bitrate"),
		Misc:           p.parseString(18, "misc"),
	}

	return streamEntry, p.errs()
}

type parser struct {
	parts  []string
	errors []error
}

func (p *parser) parseString(index int, field string) string {

	if len(p.parts) <= index {
		p.errors = append(p.errors, fmt.Errorf("parsing %s", field))
		return ""
	}

	return p.parts[index]
}

func (p *parser) parseFloat32(index int, field string) float32 {
	if len(p.parts) <= index {
		p.errors = append(p.errors, fmt.Errorf("parsing %s", field))
		return 0
	}

	floatField, err := strconv.ParseFloat(p.parts[index], 64)
	if err != nil {
		p.errors = append(p.errors, fmt.Errorf("converting %s to a float32", field))
		return 0
	}

	return float32(floatField)
}

func (p *parser) parseInt(index int, field string) int {
	if len(p.parts) <= index {
		p.errors = append(p.errors, fmt.Errorf("parsing %s", field))
		return 0
	}

	floatField, err := strconv.ParseInt(p.parts[index], 10, 64)
	if err != nil {
		p.errors = append(p.errors, fmt.Errorf("converting %s to an int", field))
		return 0
	}

	return int(floatField)
}

func (p *parser) parseBool(index int, falseValue string, field string) bool {
	if len(p.parts) <= index {
		p.errors = append(p.errors, fmt.Errorf("parsing %s", field))
		return false
	}

	val := true
	if p.parts[index] == falseValue {
		val = false
	}

	return val
}

func (p *parser) errs() []error {
	return p.errors
}
