package ntrip

import (
	"fmt"
)

// TODO: Sourcetable type

// Mount represents a NTRIP mountpoint
type Mount struct {
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
	// TODO: Authentication type - spec says: B, D, N or a comma separated list of these
	Authentication string
	Fee            bool
	Bitrate        int
	Misc           string
}

// String representation of Mount in NTRIP Sourcetable entry format
func (m Mount) String() string {
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
