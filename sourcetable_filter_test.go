package ntrip

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSourcetableFilter(t *testing.T) {
	// Create a test sourcetable
	st := Sourcetable{
		Casters: []CasterEntry{
			{
				Host:                "caster1.example.com",
				Port:                2101,
				Identifier:          "CASTER1",
				Operator:            "Example Operator",
				NMEA:                false,
				Country:             "DEU",
				Latitude:            50.1,
				Longitude:           8.5,
				FallbackHostAddress: "fallback.example.com",
				FallbackHostPort:    2101,
				Misc:                "misc1",
			},
			{
				Host:                "caster2.example.com",
				Port:                2101,
				Identifier:          "CASTER2",
				Operator:            "Example Operator",
				NMEA:                true,
				Country:             "USA",
				Latitude:            37.7,
				Longitude:           -122.4,
				FallbackHostAddress: "fallback.example.com",
				FallbackHostPort:    2101,
				Misc:                "misc2",
			},
		},
		Networks: []NetworkEntry{
			{
				Identifier:          "NET1",
				Operator:            "Example Operator",
				Authentication:      "B",
				Fee:                 false,
				NetworkInfoURL:      "http://example.com/net1",
				StreamInfoURL:       "http://example.com/net1/streams",
				RegistrationAddress: "register@example.com",
				Misc:                "misc1",
			},
			{
				Identifier:          "NET2",
				Operator:            "Example Operator",
				Authentication:      "N",
				Fee:                 true,
				NetworkInfoURL:      "http://example.com/net2",
				StreamInfoURL:       "http://example.com/net2/streams",
				RegistrationAddress: "register@example.com",
				Misc:                "misc2",
			},
		},
		Mounts: []StreamEntry{
			{
				Name:           "MOUNT1",
				Identifier:     "MOUNT1",
				Format:         "RTCM 3.2",
				FormatDetails:  "1004(1),1005(5),1006(5),1008(5),1012(1),1013(5),1033(5)",
				Carrier:        "2",
				NavSystem:      "GPS+GLO",
				Network:        "NET1",
				CountryCode:    "DEU",
				Latitude:       50.09,
				Longitude:      8.66,
				NMEA:           false,
				Solution:       false,
				Generator:      "TRIMBLE NetR9",
				Compression:    "none",
				Authentication: "B",
				Fee:            false,
				Bitrate:        9600,
				Misc:           "misc1",
			},
			{
				Name:           "MOUNT2",
				Identifier:     "MOUNT2",
				Format:         "RTCM 3.3",
				FormatDetails:  "1004(1),1005(5),1006(5),1008(5),1012(1),1013(5),1033(5)",
				Carrier:        "2",
				NavSystem:      "GPS+GLO+GAL",
				Network:        "NET1",
				CountryCode:    "USA",
				Latitude:       37.7,
				Longitude:      -122.4,
				NMEA:           true,
				Solution:       true,
				Generator:      "TRIMBLE NetR9",
				Compression:    "none",
				Authentication: "N",
				Fee:            true,
				Bitrate:        4800,
				Misc:           "misc2",
			},
		},
	}

	// Test cases
	tests := []struct {
		name             string
		query            string
		expectedCasters  int
		expectedNetworks int
		expectedMounts   int
		expectError      bool
	}{
		{
			name:             "Empty query",
			query:            "",
			expectedCasters:  2,
			expectedNetworks: 2,
			expectedMounts:   2,
			expectError:      false,
		},
		{
			name:             "Filter by country",
			query:            "?STR;;;;;;;;DEU",
			expectedCasters:  0,
			expectedNetworks: 0,
			expectedMounts:   1,
			expectError:      false,
		},
		{
			name:             "Filter by country with explicit field",
			query:            "?Country=DEU",
			expectedCasters:  1,
			expectedNetworks: 0,
			expectedMounts:   0,
			expectError:      false,
		},
		{
			name:             "Filter by bitrate",
			query:            "?Bitrate>5000",
			expectedCasters:  0,
			expectedNetworks: 0,
			expectedMounts:   1,
			expectError:      false,
		},
		{
			name:             "Filter by authentication",
			query:            "?Authentication=N",
			expectedCasters:  0,
			expectedNetworks: 1,
			expectedMounts:   1,
			expectError:      false,
		},
		{
			name:             "Filter by multiple conditions",
			query:            "?Country=USA&NMEA=true",
			expectedCasters:  1,
			expectedNetworks: 0,
			expectedMounts:   0,
			expectError:      false,
		},
		{
			name:             "Filter with no matches",
			query:            "?CountryCode=FRA",
			expectedCasters:  0,
			expectedNetworks: 0,
			expectedMounts:   0,
			expectError:      false,
		},
		{
			name:             "Filter with contains operator",
			query:            "?NavSystem~GAL",
			expectedCasters:  0,
			expectedNetworks: 0,
			expectedMounts:   1,
			expectError:      false,
		},
		{
			name:             "Invalid query format",
			query:            "?invalid-query",
			expectedCasters:  2,
			expectedNetworks: 2,
			expectedMounts:   2,
			expectError:      true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			filtered, err := st.Filter(tc.query)

			if tc.expectError {
				require.Error(t, err)
				// On error, the original sourcetable should be returned
				require.Equal(t, len(st.Casters), len(filtered.Casters))
				require.Equal(t, len(st.Networks), len(filtered.Networks))
				require.Equal(t, len(st.Mounts), len(filtered.Mounts))
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedCasters, len(filtered.Casters))
				require.Equal(t, tc.expectedNetworks, len(filtered.Networks))
				require.Equal(t, tc.expectedMounts, len(filtered.Mounts))
			}
		})
	}
}

func TestParseQuery(t *testing.T) {
	tests := []struct {
		name               string
		queryStr           string
		expectedConditions int
		expectError        bool
	}{
		{
			name:               "Empty query",
			queryStr:           "",
			expectedConditions: 0,
			expectError:        false,
		},
		{
			name:               "Simple field=value query",
			queryStr:           "?CountryCode=DEU",
			expectedConditions: 1,
			expectError:        false,
		},
		{
			name:               "Multiple conditions",
			queryStr:           "?CountryCode=DEU&Bitrate>5000",
			expectedConditions: 2,
			expectError:        false,
		},
		{
			name:               "NTRIP format with semicolons",
			queryStr:           "?STR;;;;;;;;DEU",
			expectedConditions: 1,
			expectError:        false,
		},
		{
			name:               "NTRIP format with multiple fields",
			queryStr:           "?STR;MOUNT1;;;;;;;DEU",
			expectedConditions: 2,
			expectError:        false,
		},
		{
			name:               "Invalid operator",
			queryStr:           "?&CountryCode@DEU",
			expectedConditions: 0,
			expectError:        true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			q, err := parseQuery(tc.queryStr)

			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedConditions, len(q.conditions))
			}
		})
	}
}

func TestMatchesCondition(t *testing.T) {
	// Test struct
	type testStruct struct {
		StringField string
		IntField    int
		FloatField  float64
		BoolField   bool
	}

	// Test instance
	testInstance := testStruct{
		StringField: "test",
		IntField:    42,
		FloatField:  3.14,
		BoolField:   true,
	}

	// Test cases
	tests := []struct {
		name      string
		condition condition
		expected  bool
	}{
		{
			name: "String equals",
			condition: condition{
				field:    "StringField",
				operator: "=",
				value:    "test",
			},
			expected: true,
		},
		{
			name: "String not equals",
			condition: condition{
				field:    "StringField",
				operator: "!=",
				value:    "other",
			},
			expected: true,
		},
		{
			name: "String contains",
			condition: condition{
				field:    "StringField",
				operator: "~",
				value:    "es",
			},
			expected: true,
		},
		{
			name: "Int equals",
			condition: condition{
				field:    "IntField",
				operator: "=",
				value:    "42",
			},
			expected: true,
		},
		{
			name: "Int greater than",
			condition: condition{
				field:    "IntField",
				operator: ">",
				value:    "30",
			},
			expected: true,
		},
		{
			name: "Int less than",
			condition: condition{
				field:    "IntField",
				operator: "<",
				value:    "50",
			},
			expected: true,
		},
		{
			name: "Float equals",
			condition: condition{
				field:    "FloatField",
				operator: "=",
				value:    "3.14",
			},
			expected: true,
		},
		{
			name: "Bool equals",
			condition: condition{
				field:    "BoolField",
				operator: "=",
				value:    "true",
			},
			expected: true,
		},
		{
			name: "Non-existent field",
			condition: condition{
				field:    "NonExistentField",
				operator: "=",
				value:    "value",
			},
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := matchesCondition(testInstance, tc.condition)
			require.Equal(t, tc.expected, result)
		})
	}
}
