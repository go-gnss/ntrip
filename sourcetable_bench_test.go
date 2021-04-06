package ntrip_test

import (
	"fmt"
	"testing"

	"github.com/go-gnss/ntrip"
)

func BenchmarkSourcetableString(b *testing.B) {

	sourcetable := ntrip.Sourcetable{}

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
