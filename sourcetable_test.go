package ntrip

import (
	"fmt"
	"testing"
)

var (
	sourcetable Sourcetable = Sourcetable{
		Casters: []CasterEntry{
			{
				Host:                "host",
				Port:                2101,
				Identifier:          "identifier",
				Operator:            "operator",
				NMEA:                false,
				Country:             "AUS",
				Latitude:            0.1,
				Longitude:           -0.1,
				FallbackHostAddress: "fallback",
				FallbackHostPort:    12101,
				Misc:                "misc",
			},
			{
				Host:                "host2",
				Port:                2102,
				Identifier:          "identifier2",
				Operator:            "operator2",
				NMEA:                true,
				Country:             "AUS",
				Latitude:            -0.1,
				Longitude:           0.1,
				FallbackHostAddress: "fallback2",
				FallbackHostPort:    12102,
				Misc:                "misc2",
			},
		},
		Networks: []NetworkEntry{
			{
				Identifier:          "identifier",
				Operator:            "operator",
				Authentication:      "B",
				Fee:                 false,
				NetworkInfoURL:      "https://network.info",
				StreamInfoURL:       "https://stream.info",
				RegistrationAddress: "register@operator.io",
				Misc:                "misc",
			},
			{
				Identifier:          "identifier2",
				Operator:            "operator2",
				Authentication:      "N",
				Fee:                 true,
				NetworkInfoURL:      "https://network2.info",
				StreamInfoURL:       "https://stream2.info",
				RegistrationAddress: "register2@operator.io",
				Misc:                "misc2",
			},
		},
		Mounts: []MountEntry{
			{
				Name:           "name",
				Identifier:     "identifier",
				Format:         "format",
				FormatDetails:  "format details",
				Carrier:        "carrier",
				NavSystem:      "nav system",
				Network:        "network",
				CountryCode:    "AUS",
				Latitude:       1.0,
				Longitude:      -1.0,
				NMEA:           false,
				Solution:       false,
				Generator:      "generator",
				Compression:    "compression",
				Authentication: "N",
				Fee:            false,
				Bitrate:        0,
				Misc:           "misc",
			},
			{
				Name:           "name2",
				Identifier:     "identifier2",
				Format:         "format2",
				FormatDetails:  "format details2",
				Carrier:        "carrier2",
				NavSystem:      "nav system2",
				Network:        "network2",
				CountryCode:    "AUS",
				Latitude:       2.0,
				Longitude:      -2.0,
				NMEA:           true,
				Solution:       true,
				Generator:      "generator2",
				Compression:    "compression2",
				Authentication: "B",
				Fee:            true,
				Bitrate:        0,
				Misc:           "misc2",
			},
		},
	}
	sourcetableString string = fmt.Sprintf("%s\r\n%s\r\n%s\r\n%s\r\n%s\r\n%s",
		"CAS;host;2101;identifier;operator;0;AUS;0.1000;-0.1000;fallback;12101;misc",
		"CAS;host2;2102;identifier2;operator2;1;AUS;-0.1000;0.1000;fallback2;12102;misc2",
		"NET;identifier;operator;B;N;https://network.info;https://stream.info;register@operator.io;misc",
		"NET;identifier2;operator2;N;Y;https://network2.info;https://stream2.info;register2@operator.io;misc2",
		"STR;name;identifier;format;format details;carrier;nav system;network;AUS;1.0000;-1.0000;0;0;generator;compression;N;N;0;misc",
		"STR;name2;identifier2;format2;format details2;carrier2;nav system2;network2;AUS;2.0000;-2.0000;1;1;generator2;compression2;B;Y;0;misc2")
)

func TestSourcetableString(t *testing.T) {
	if sourcetable.String() != sourcetableString {
		t.Errorf("sourcetable did not convert to string correctly")
	}
}
