package ntrip

import (
	"fmt"
)

// Sourcetable for NTRIP Casters, returned at / as a way for users to discover available mounts
type Sourcetable struct {
	Casters  []CasterEntry
	Networks []NetworkEntry
	Mounts   []MountEntry
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
	Latitude      string
	Longitude     string
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

	return fmt.Sprintf("STR;%s;%s;%s;%s;%s;%s;%s;%s;%s;%s;%s;%s;%s;%s;%s;%s;%d;%s",
		m.Name, m.Identifier, m.Format, m.FormatDetails, m.Carrier, m.NavSystem, m.Network,
		m.CountryCode, m.Latitude, m.Longitude, nmea, solution, m.Generator, m.Compression,
		m.Authentication, fee, m.Bitrate, m.Misc)
}
