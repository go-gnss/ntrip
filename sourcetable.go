package ntrip

import (
	"context"
	"fmt"
	"io"
	"reflect"
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

// Filter filters the sourcetable based on a query string
// Format: ?STR;;;;;;DEU&latitude>50.0&bitrate~9600
func (st Sourcetable) Filter(query string) (Sourcetable, error) {
	if query == "" {
		return st, nil
	}

	result := Sourcetable{
		Casters:  []CasterEntry{},
		Networks: []NetworkEntry{},
		Mounts:   []StreamEntry{},
	}

	// Parse the query
	parsedQuery, err := parseQuery(query)
	if err != nil {
		return st, err
	}

	// If no conditions, return the original sourcetable
	if len(parsedQuery.conditions) == 0 {
		return st, nil
	}

	// Filter casters
	for _, caster := range st.Casters {
		if parsedQuery.matches(caster) {
			result.Casters = append(result.Casters, caster)
		}
	}

	// Filter networks
	for _, network := range st.Networks {
		if parsedQuery.matches(network) {
			result.Networks = append(result.Networks, network)
		}
	}

	// Filter mounts
	for _, mount := range st.Mounts {
		if parsedQuery.matches(mount) {
			result.Mounts = append(result.Mounts, mount)
		}
	}

	return result, nil
}

// query represents a parsed filter query
type query struct {
	conditions []condition
}

// condition represents a single filter condition
type condition struct {
	field    string
	operator string
	value    string
}

// parseQuery parses a query string into a query object
func parseQuery(queryStr string) (query, error) {
	q := query{conditions: []condition{}}

	if queryStr == "" || !strings.HasPrefix(queryStr, "?") {
		return q, nil
	}

	// Remove the leading '?'
	queryStr = queryStr[1:]

	parts := strings.Split(queryStr, "&")
	for i, part := range parts {
		// The first part is special and uses semicolons to separate fields
		if i == 0 && strings.Contains(part, ";") {
			fields := strings.Split(part, ";")
			if len(fields) > 0 && fields[0] != "" {
				// First field is the entry type (STR, CAS, NET)
				entryType := fields[0]

				// Add conditions for non-empty fields
				for j, field := range fields[1:] {
					if field == "" {
						continue
					}

					// Map field index to actual field name based on entry type
					fieldName := getFieldNameByIndex(entryType, j)
					if fieldName != "" {
						q.conditions = append(q.conditions, condition{
							field:    fieldName,
							operator: "=",
							value:    field,
						})
					}
				}
			}
		} else {
			// Regular condition with operator
			var op string
			var idx int

			// Check for each operator
			for _, operator := range []string{"!=", ">=", "<=", "=", ">", "<", "~"} {
				if i := strings.Index(part, operator); i >= 0 {
					op = operator
					idx = i
					break
				}
			}

			if op == "" {
				return q, fmt.Errorf("invalid condition format: %s", part)
			}

			field := part[:idx]
			value := part[idx+len(op):]

			q.conditions = append(q.conditions, condition{
				field:    field,
				operator: op,
				value:    value,
			})
		}
	}

	return q, nil
}

// getFieldNameByIndex maps field indices to field names based on entry type
func getFieldNameByIndex(entryType string, index int) string {
	switch entryType {
	case "STR":
		fields := []string{
			"Name", "Identifier", "Format", "FormatDetails", "Carrier",
			"NavSystem", "Network", "CountryCode", "Latitude", "Longitude",
			"NMEA", "Solution", "Generator", "Compression", "Authentication",
			"Fee", "Bitrate", "Misc",
		}
		if index < len(fields) {
			return fields[index]
		}
	case "CAS":
		fields := []string{
			"Host", "Port", "Identifier", "Operator", "NMEA",
			"Country", "Latitude", "Longitude", "FallbackHostAddress", "FallbackHostPort",
			"Misc",
		}
		if index < len(fields) {
			return fields[index]
		}
	case "NET":
		fields := []string{
			"Identifier", "Operator", "Authentication", "Fee", "NetworkInfoURL",
			"StreamInfoURL", "RegistrationAddress", "Misc",
		}
		if index < len(fields) {
			return fields[index]
		}
	}
	return ""
}

// matches checks if an entry matches the query
func (q query) matches(entry interface{}) bool {
	if len(q.conditions) == 0 {
		return true
	}

	for _, cond := range q.conditions {
		if !matchesCondition(entry, cond) {
			return false
		}
	}

	return true
}

// matchesCondition checks if an entry matches a single condition
func matchesCondition(entry interface{}, cond condition) bool {
	val := reflect.ValueOf(entry)

	// If it's a pointer, get the value it points to
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	// Only struct types are supported
	if val.Kind() != reflect.Struct {
		return false
	}

	// Find the field
	field := val.FieldByName(cond.field)
	if !field.IsValid() {
		return false
	}

	// Convert field value to string for comparison
	var fieldStr string
	switch field.Kind() {
	case reflect.String:
		fieldStr = field.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		fieldStr = strconv.FormatInt(field.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		fieldStr = strconv.FormatUint(field.Uint(), 10)
	case reflect.Float32, reflect.Float64:
		fieldStr = strconv.FormatFloat(field.Float(), 'f', -1, 64)
	case reflect.Bool:
		fieldStr = strconv.FormatBool(field.Bool())
	default:
		return false
	}

	// Compare based on operator
	switch cond.operator {
	case "=":
		return fieldStr == cond.value
	case "!=":
		return fieldStr != cond.value
	case "~":
		return strings.Contains(fieldStr, cond.value)
	case ">", ">=", "<", "<=":
		// Parse numbers for comparison
		fieldVal, err1 := strconv.ParseFloat(fieldStr, 64)
		condVal, err2 := strconv.ParseFloat(cond.value, 64)

		if err1 != nil || err2 != nil {
			// Fall back to string comparison if not numeric
			switch cond.operator {
			case ">":
				return fieldStr > cond.value
			case ">=":
				return fieldStr >= cond.value
			case "<":
				return fieldStr < cond.value
			case "<=":
				return fieldStr <= cond.value
			}
		}

		switch cond.operator {
		case ">":
			return fieldVal > condVal
		case ">=":
			return fieldVal >= condVal
		case "<":
			return fieldVal < condVal
		case "<=":
			return fieldVal <= condVal
		}
	}

	return false
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
// The function returns a list of errors which can be treated as warnings.
// These warnings indicate that the caster is returning an improper rtcm3 format.
func GetSourcetable(ctx context.Context, url string) (Sourcetable, []error, error) {
	warnings := []error{}

	// Create a request using the provided context
	req, err := NewClientRequestWithContext(ctx, url)
	if err != nil {
		return Sourcetable{}, warnings, errors.Wrap(err, "building request")
	}

	// Use the properly configured client
	client := DefaultHTTPClient()

	// Make the request
	res, err := client.Do(req)
	if err != nil {
		return Sourcetable{}, warnings, err
	}
	defer res.Body.Close()

	// Read the response body
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return Sourcetable{}, warnings, err
	}

	if res.StatusCode != 200 {
		return Sourcetable{}, warnings, fmt.Errorf("received a non 200 status code: %d", res.StatusCode)
	}

	// Swallowing the errors here is okay because the errors are more like warnings.
	// All rows that could be parsed will be present in the source table.
	table, warnings := ParseSourcetable(string(body))
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
