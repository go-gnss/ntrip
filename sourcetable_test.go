package ntrip

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/gobuffalo/httptest"
	"github.com/stretchr/testify/require"
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
		Mounts: []StreamEntry{
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
	sourcetableString string = fmt.Sprintf("%s\r\n%s\r\n%s\r\n%s\r\n%s\r\n%s\r\n%s\r\n",
		"CAS;host;2101;identifier;operator;0;AUS;0.1000;-0.1000;fallback;12101;misc",
		"CAS;host2;2102;identifier2;operator2;1;AUS;-0.1000;0.1000;fallback2;12102;misc2",
		"NET;identifier;operator;B;N;https://network.info;https://stream.info;register@operator.io;misc",
		"NET;identifier2;operator2;N;Y;https://network2.info;https://stream2.info;register2@operator.io;misc2",
		"STR;name;identifier;format;format details;carrier;nav system;network;AUS;1.0000;-1.0000;0;0;generator;compression;N;N;0;misc",
		"STR;name2;identifier2;format2;format details2;carrier2;nav system2;network2;AUS;2.0000;-2.0000;1;1;generator2;compression2;B;Y;0;misc2",
		"ENDSOURCETABLE",
	)
)

func TestSourcetableString(t *testing.T) {
	if sourcetable.String() != sourcetableString {
		t.Errorf("sourcetable did not convert to string correctly")
	}
}

func TestDecodeSourcetable(t *testing.T) {

	// Arrange
	var (
		table = `
		CAS;auscors.ga.gov.au;2101;AUSCORS Ntrip Broadcaster;GA;0;AUS;-35.34;149.18;http://something;5454;misc
		CAS;rtcm-ntrip.org;2101;NtripInfoCaster;BKG;0;DEU;50.12;8.69;0.0.0.0;0;http://www.rtcm-ntrip.org/home
		NET;ARGN;GA;B;N;http://www.ga.gov.au;https://gws.geodesy.ga.gov.au/skeletonFiles/;gnss@ga.gov.au;xyz
		NET;AUSCOPE;GA;B;N;http://www.ga.gov.au;https://gws.geodesy.ga.gov.au/skeletonFiles/;gnss@ga.gov.au;
		NET;SPRGN;GA;B;N;http://www.ga.gov.au;https://gws.geodesy.ga.gov.au/skeletonFiles/;gnss@ga.gov.au;
		NET;APREF;GA;B;N;http://www.ga.gov.au;https://gws.geodesy.ga.gov.au/skeletonFiles/;gnss@ga.gov.au;
		NET;IGS;IGS;B;N;https://igs.bkg.bund.de/root_ftp/NTRIP/streams/streamlist_igs-ip.htm;https://igs.bkg.bund.de:443/root_ftp/MGEX/station/rnxskl/;http://register.rtcm-ntrip.org;none
		STR;31NA00AUS0;Alice Springs AZRI (NT);RTCM 3.2;1006(10),1013(10),1019(60),1020(60),1033(10),1042(60),1044(60),1046(60),1077(1),1087(1),1097(1),1117(1),1127(1),1230(10);2;GPS+GLO+GAL+BDS+QZS;APREF;AUS;-23.76698;133.87921;0;0;SEPT POLARX4TR;none;B;N;9600;DLP
		STR;SWTC00AUS0;Spring Mountain (QLD);RTCM 3.2;1006(10),1013(10),1019(60),1020(60),1033(10),1042(60),1044(60),1046(60),1077(1),1087(1),1097(1),1117(1),1127(1),1230(10);0;GPS+GLO+GAL+BDS+QZS;APREF;AUS;-27.67242;152.87707;0;0;Leica GR30;none;B;N;9600;TMI
		STR;CA1000AUS0;Mount Annan (NSW);RTCM 3.2;1006(10),1013(10),1019(60),1020(60),1033(10),1042(60),1044(60),1046(60),1077(1),1087(1),1097(1),1117(1),1127(1),1230(10);2;GPS+GLO+GAL+BDS+QZS;ARGN;AUS;-34.07194;150.7602;0;0;TRIMBLE NETR9;none;B;N;9600;GA
		STR;ALBY00AUS0;Albany (WA);RTCM 3.2;1006(10),1013(10),1019(60),1020(60),1033(10),1042(60),1044(60),1046(60),1077(1),1087(1),1097(1),1117(1),1127(1),1230(10);2;GPS+GLO+GAL+BDS+QZS;AUSCOPE;AUS;-34.95023;117.81018;0;0;SEPT POLARX5;none;B;N;9600;Landgate
		STR;ALIC00AUS0;Alice Springs (NT);RTCM 3.2;1006(10),1013(10),1019(60),1020(60),1033(10),1042(60),1044(60),1046(60),1077(1),1087(1),1097(1),1117(1),1127(1),1230(10);2;GPS+GLO+GAL+BDS+QZS;ARGN;AUS;-23.67012;133.88551;0;0;SEPT POLARX5;none;B;N;9600;GA
		STR;ANDA00AUS0;Andamooka (SA);RTCM 3.2;1006(10),1013(10),1019(60),1020(60),1033(10),1042(60),1044(60),1046(60),1077(1),1087(1),1097(1),1117(1),1127(1),1230(10);2;GPS+GLO+GAL+BDS+QZS;AUSCOPE;AUS;-30.4533;137.1601;0;0;LEICA GR30;none;B;N;9600;DPTI
		STR;ARMC00AUS0;Aramac (QLD);RTCM 3.2;1006(10),1013(10),1019(60),1020(60),1033(10),1042(60),1044(60),1046(60),1077(1),1087(1),1097(1),1117(1),1127(1),1230(10);2;GPS+GLO+GAL+BDS+QZS;AUSCOPE;AUS;-22.95682;145.24545;0;0;TRIMBLE NETR9;none;B;N;9600;DNRM
		STR;MAJU00MHL0;Majuro;RTCM 3.2;1006(10),1013(10),1019(60),1020(60),1033(10),1042(60),1044(60),1046(60),1077(1),1087(1),1097(1),1117(1),1127(1),1230(10);2;GPS+GLO+GAL+BDS+QZS;SPRGN;MHL;7.11914;171.36452;0;0;SEPT POLARX4TR;none;B;N;9600;GA
		STR;RTCM3EPH-MGEX;Assisted-GNSS;RTCM 3.3;1019,1020,1042,1043,1044,1045,1046;0;GPS+GLO+GAL+BDS+QZS+SBAS;IGS;DEU;50.09;8.66;0;1;euronet;none;B;N;3600;BKG
		ENDSOURCETABLE
		`
	)

	// Act
	sourcetable, err := ParseSourcetable(table)

	// Assert
	// should report the three 'errors' from the first caster not matching the spec
	require.Len(t, err, 0, "error decoding source table")

	// Assert Casters
	require.Len(t, sourcetable.Casters, 2, "wrong number of casters")
	require.Equal(t, "auscors.ga.gov.au", sourcetable.Casters[0].Host)
	require.Equal(t, 2101, sourcetable.Casters[0].Port)
	require.Equal(t, "AUSCORS Ntrip Broadcaster", sourcetable.Casters[0].Identifier)
	require.Equal(t, "GA", sourcetable.Casters[0].Operator)
	require.Equal(t, false, sourcetable.Casters[0].NMEA)
	require.Equal(t, "AUS", sourcetable.Casters[0].Country)
	require.Equal(t, float32(-35.34), sourcetable.Casters[0].Latitude)
	require.Equal(t, float32(149.18), sourcetable.Casters[0].Longitude)
	require.Equal(t, "http://something", sourcetable.Casters[0].FallbackHostAddress)
	require.Equal(t, 5454, sourcetable.Casters[0].FallbackHostPort)
	require.Equal(t, "misc", sourcetable.Casters[0].Misc)

	require.Equal(t, "0.0.0.0", sourcetable.Casters[1].FallbackHostAddress)
	require.Equal(t, 0, sourcetable.Casters[1].FallbackHostPort)
	require.Equal(t, "http://www.rtcm-ntrip.org/home", sourcetable.Casters[1].Misc)

	// Assert Networks
	require.Len(t, sourcetable.Networks, 5, "wrong number of networks")
	require.Equal(t, "ARGN", sourcetable.Networks[0].Identifier)
	require.Equal(t, "GA", sourcetable.Networks[0].Operator)
	require.Equal(t, "B", sourcetable.Networks[0].Authentication)
	require.Equal(t, false, sourcetable.Networks[0].Fee)
	require.Equal(t, "http://www.ga.gov.au", sourcetable.Networks[0].NetworkInfoURL)
	require.Equal(t, "https://gws.geodesy.ga.gov.au/skeletonFiles/", sourcetable.Networks[0].StreamInfoURL)
	require.Equal(t, "gnss@ga.gov.au", sourcetable.Networks[0].RegistrationAddress)
	require.Equal(t, "xyz", sourcetable.Networks[0].Misc)

	// Assert Mount
	// STR;31NA00AUS0;Alice Springs AZRI (NT);RTCM 3.2;1006(10),1013(10),1019(60),1020(60),1033(10),1042(60),1044(60),1046(60),1077(1),1087(1),1097(1),1117(1),1127(1),1230(10);2;GPS+GLO+GAL+BDS+QZS;APREF;AUS;-23.76698;133.87921;0;0;SEPT POLARX4TR;none;B;N;9600;DLP
	require.Len(t, sourcetable.Mounts, 9, "wrong number of mounts")
	require.Equal(t, "31NA00AUS0", sourcetable.Mounts[0].Name)
	require.Equal(t, "Alice Springs AZRI (NT)", sourcetable.Mounts[0].Identifier)
	require.Equal(t, "RTCM 3.2", sourcetable.Mounts[0].Format)
	require.Equal(t, "1006(10),1013(10),1019(60),1020(60),1033(10),1042(60),1044(60),1046(60),1077(1),1087(1),1097(1),1117(1),1127(1),1230(10)", sourcetable.Mounts[0].FormatDetails)
	require.Equal(t, "2", sourcetable.Mounts[0].Carrier)
	require.Equal(t, "GPS+GLO+GAL+BDS+QZS", sourcetable.Mounts[0].NavSystem)
	require.Equal(t, "APREF", sourcetable.Mounts[0].Network)
	require.Equal(t, "AUS", sourcetable.Mounts[0].CountryCode)
	require.InDelta(t, -23.76698, float64(sourcetable.Mounts[0].Latitude), 0.0001)
	require.InDelta(t, 133.87921, float64(sourcetable.Mounts[0].Longitude), 0.0001)
	require.Equal(t, false, sourcetable.Mounts[0].NMEA)
	require.Equal(t, false, sourcetable.Mounts[0].Solution)
	require.Equal(t, "SEPT POLARX4TR", sourcetable.Mounts[0].Generator)
	require.Equal(t, "none", sourcetable.Mounts[0].Compression)
	require.Equal(t, 9600, sourcetable.Mounts[0].Bitrate)
	require.Equal(t, "DLP", sourcetable.Mounts[0].Misc)
}

func TestGetSourcetable(t *testing.T) {

	// Arrange
	ctx := context.Background()

	table := `
		CAS;auscors.ga.gov.au;2101;AUSCORS Ntrip Broadcaster;GA;0;AUS;-35.34;149.18
		NET;ARGN;GA;B;N;http://www.ga.gov.au;https://gws.geodesy.ga.gov.au/skeletonFiles/;gnss@ga.gov.au;
		NET;IGS;IGS;B;N;https://igs.bkg.bund.de/root_ftp/NTRIP/streams/streamlist_igs-ip.htm;https://igs.bkg.bund.de:443/root_ftp/MGEX/station/rnxskl/;http://register.rtcm-ntrip.org;none
		STR;31NA00AUS0;Alice Springs AZRI (NT);RTCM 3.2;1006(10),1013(10),1019(60),1020(60),1033(10),1042(60),1044(60),1046(60),1077(1),1087(1),1097(1),1117(1),1127(1),1230(10);2;GPS+GLO+GAL+BDS+QZS;APREF;AUS;-23.76698;133.87921;0;0;SEPT POLARX4TR;none;B;N;9600;DLP
		STR;ALBY00AUS0;Albany (WA);RTCM 3.2;1006(10),1013(10),1019(60),1020(60),1033(10),1042(60),1044(60),1046(60),1077(1),1087(1),1097(1),1117(1),1127(1),1230(10);2;GPS+GLO+GAL+BDS+QZS;AUSCOPE;AUS;-34.95023;117.81018;0;0;SEPT POLARX5;none;B;N;9600;Landgate
		STR;ALIC00AUS0;Alice Springs (NT);RTCM 3.2;1006(10),1013(10),1019(60),1020(60),1033(10),1042(60),1044(60),1046(60),1077(1),1087(1),1097(1),1117(1),1127(1),1230(10);2;GPS+GLO+GAL+BDS+QZS;ARGN;AUS;-23.67012;133.88551;0;0;SEPT POLARX5;none;B;N;9600;GA
		ENDSOURCETABLE
		`

	handler := func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, table)
	}

	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	// Act
	mapping, warnings, err := GetSourcetable(ctx, server.URL)

	// Assert
	require.Nil(t, err, "got error getting sourcetable")
	require.Len(t, warnings, 3, "got improper number of warnings")
	expected, _ := ParseSourcetable(table)
	require.Equal(t, expected, mapping)
}
