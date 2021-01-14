package ntrip

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// Sourcetable for NTRIP Casters, returned at / as a way for users to discover available mounts
type Sourcetable struct {
	Casters  []CasterEntry
	Networks []NetworkEntry
	Mounts   []MountEntry
}

func (st Sourcetable) String() (s string) {
	for _, cas := range st.Casters {
		s = fmt.Sprintf("%s%s\r\n", s, cas)
	}

	for _, net := range st.Networks {
		s = fmt.Sprintf("%s%s\r\n", s, net)
	}

	for _, str := range st.Mounts {
		s = fmt.Sprintf("%s%s\r\n", s, str)
	}

	return s + "ENDSOURCETABLE\r\n"
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

	return fmt.Sprintf("CAS;%s;%d;%s;%s;%s;%s;%.4f;%.4f;%s;%d;%s",
		c.Host, c.Port, c.Identifier, c.Operator, nmea, c.Country, c.Latitude, c.Longitude,
		c.FallbackHostAddress, c.FallbackHostPort, c.Misc)
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

	return fmt.Sprintf("NET;%s;%s;%s;%s;%s;%s;%s;%s",
		n.Identifier, n.Operator, n.Authentication, fee, n.NetworkInfoURL, n.StreamInfoURL,
		n.RegistrationAddress, n.Misc)
}

// MountEntry for an NTRIP Sourcetable
type MountEntry struct {
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
func (m MountEntry) String() string {
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

	return fmt.Sprintf("STR;%s;%s;%s;%s;%s;%s;%s;%s;%.4f;%.4f;%s;%s;%s;%s;%s;%s;%d;%s",
		m.Name, m.Identifier, m.Format, m.FormatDetails, m.Carrier, m.NavSystem, m.Network,
		m.CountryCode, m.Latitude, m.Longitude, nmea, solution, m.Generator, m.Compression,
		m.Authentication, fee, m.Bitrate, m.Misc)
}

// ParseSourcetable parses a sourcetable from an ioreader into a ntrip style source table.
func ParseSourcetable(str string) (Sourcetable, []error) {
	table := Sourcetable{}
	var errs []error

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
			caster, err := ParseCasterEntry(line)
			if err != nil {
				errs = append(errs, errors.Wrapf(err, "parsing caster %v", lineNo))
			}
			table.Casters = append(table.Casters, caster)
		case "NET":
			net, err := ParseNetworkEntry(line)
			if err != nil {
				errs = append(errs, errors.Wrapf(err, "parsing line %v", lineNo))
			}
			table.Networks = append(table.Networks, net)
		case "STR":
			mount, err := ParseStreamEntry(line)
			if err != nil {
				errs = append(errs, errors.Wrapf(err, "parsing line %v", lineNo))
			}
			table.Mounts = append(table.Mounts, mount)
		}

	}

	return table, errs
}

// ParseCasterEntry parses a single caster from a string.
func ParseCasterEntry(casterString string) (CasterEntry, error) {
	parts := strings.Split(casterString, ";")

	long, err := strconv.ParseFloat(parts[8], 64)
	if err != nil {
		return CasterEntry{}, fmt.Errorf("invalid longitude")
	}

	lat, err := strconv.ParseFloat(parts[7], 64)
	if err != nil {
		fmt.Println(err)
		return CasterEntry{}, fmt.Errorf("invalid latitude")
	}

	nmea := true
	if parts[5] == "0" {
		nmea = false
	}

	port, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		fmt.Println(err)
		return CasterEntry{}, fmt.Errorf("invalid port")
	}

	return CasterEntry{
		Host:       parts[1],
		Port:       int(port),
		Identifier: parts[3],
		Operator:   parts[4],
		NMEA:       nmea,
		Country:    parts[6],
		Latitude:   float32(lat),
		Longitude:  float32(long),
	}, nil

}

// ParseNetworkEntry parses a single network entry from a string.
func ParseNetworkEntry(netString string) (NetworkEntry, error) {
	parts := strings.Split(netString, ";")

	fee := true
	if parts[4] == "N" {
		fee = false
	}

	return NetworkEntry{
		Identifier:          parts[1],
		Operator:            parts[2],
		Authentication:      parts[3],
		Fee:                 fee,
		NetworkInfoURL:      parts[5],
		StreamInfoURL:       parts[6],
		RegistrationAddress: parts[7],
	}, nil

}

// ParseStreamEntry parses a single mount entry.
func ParseStreamEntry(streamString string) (MountEntry, error) {
	parts := strings.Split(streamString, ";")

	lat, err := strconv.ParseFloat(parts[9], 64)
	if err != nil {
		return MountEntry{}, fmt.Errorf("invalid latitude")
	}

	lng, err := strconv.ParseFloat(parts[10], 64)
	if err != nil {
		fmt.Println(err)
		return MountEntry{}, fmt.Errorf("invalid longitude")
	}

	nmea := true
	if parts[11] == "0" {
		nmea = false
	}

	solution := true
	if parts[12] == "0" {
		solution = false
	}

	fee := true
	if parts[16] == "N" {
		fee = false
	}

	bitrate, err := strconv.ParseInt(parts[17], 10, 64)
	if err != nil {
		fmt.Println(err)
		return MountEntry{}, fmt.Errorf("invalid bitrate")
	}

	return MountEntry{
		Name:          parts[1],
		Identifier:    parts[2],
		Format:        parts[3],
		FormatDetails: parts[4],
		Carrier:       parts[5],
		NavSystem:     parts[6],
		Network:       parts[7],
		CountryCode:   parts[8],
		Latitude:      float32(lat),
		Longitude:     float32(lng),
		NMEA:          nmea,
		Solution:      solution,
		Generator:     parts[13],
		Compression:   parts[14],
		// TODO: Authentication type
		Authentication: parts[15],
		Fee:            fee,
		Bitrate:        int(bitrate),
	}, nil
}
