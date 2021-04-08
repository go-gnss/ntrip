package ntrip_test

import (
	"fmt"
	"testing"

	"github.com/go-gnss/ntrip"
)

func BenchmarkSourcetableString(b *testing.B) {

	sourcetable := ntrip.Sourcetable{
		Casters: []ntrip.CasterEntry{
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
		Networks: []ntrip.NetworkEntry{
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
	}

	for i := 0; i < 10000; i++ {
		sourcetable.Mounts = append(sourcetable.Mounts, ntrip.StreamEntry{
			Name:           fmt.Sprintf("name-%v", i),
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
		})
	}

	b.ResetTimer()

	var s string
	for i := 0; i < b.N; i++ {
		s = sourcetable.String()
	}

	_ = s
}
