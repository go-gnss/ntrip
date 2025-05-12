package filter

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/go-gnss/ntrip"
)

// Operator represents a comparison operator
type Operator string

const (
	// Equal checks if values are equal
	Equal Operator = "="
	// NotEqual checks if values are not equal
	NotEqual Operator = "!="
	// GreaterThan checks if left value is greater than right value
	GreaterThan Operator = ">"
	// LessThan checks if left value is less than right value
	LessThan Operator = "<"
	// GreaterThanOrEqual checks if left value is greater than or equal to right value
	GreaterThanOrEqual Operator = ">="
	// LessThanOrEqual checks if left value is less than or equal to right value
	LessThanOrEqual Operator = "<="
	// Contains checks if left value contains right value
	Contains Operator = "~"
)

// Condition represents a single filter condition
type Condition struct {
	Field    string
	Operator Operator
	Value    string
}

// Query represents a complete filter query with multiple conditions
type Query struct {
	Conditions []Condition
}

// ParseQuery parses a query string into a Query object
// Format: field1=value1&field2>value2&field3~value3
func ParseQuery(queryStr string) (Query, error) {
	query := Query{
		Conditions: []Condition{},
	}

	if queryStr == "" {
		return query, nil
	}

	parts := strings.Split(queryStr, "&")
	for _, part := range parts {
		// Find the operator
		var op Operator
		var idx int

		// Check for each operator
		for _, operator := range []Operator{NotEqual, GreaterThanOrEqual, LessThanOrEqual, Equal, GreaterThan, LessThan, Contains} {
			if i := strings.Index(part, string(operator)); i >= 0 {
				op = operator
				idx = i
				break
			}
		}

		if op == "" {
			return query, fmt.Errorf("invalid condition format: %s", part)
		}

		field := part[:idx]
		value := part[idx+len(op):]

		query.Conditions = append(query.Conditions, Condition{
			Field:    field,
			Operator: op,
			Value:    value,
		})
	}

	return query, nil
}

// ParseNTRIPQuery parses an NTRIP-specific query format
// Format: ?STR;;;;;;DEU&latitude>50.0&bitrate~9600
func ParseNTRIPQuery(queryStr string) (Query, error) {
	query := Query{
		Conditions: []Condition{},
	}

	if queryStr == "" || !strings.HasPrefix(queryStr, "?") {
		return query, nil
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
						query.Conditions = append(query.Conditions, Condition{
							Field:    fieldName,
							Operator: Equal,
							Value:    field,
						})
					}
				}
			}
		} else {
			// Regular condition with operator
			var op Operator
			var idx int

			// Check for each operator
			for _, operator := range []Operator{NotEqual, GreaterThanOrEqual, LessThanOrEqual, Equal, GreaterThan, LessThan, Contains} {
				if i := strings.Index(part, string(operator)); i >= 0 {
					op = operator
					idx = i
					break
				}
			}

			if op == "" {
				return query, fmt.Errorf("invalid condition format: %s", part)
			}

			field := part[:idx]
			value := part[idx+len(op):]

			query.Conditions = append(query.Conditions, Condition{
				Field:    field,
				Operator: op,
				Value:    value,
			})
		}
	}

	return query, nil
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
			"Identifier", "Operator", "Authentication", "Fee", "WebNet",
			"WebStr", "WebReg", "Misc",
		}
		if index < len(fields) {
			return fields[index]
		}
	}
	return ""
}

// Matches checks if an entry matches the query
func (q Query) Matches(entry interface{}) bool {
	if len(q.Conditions) == 0 {
		return true
	}

	for _, cond := range q.Conditions {
		if !matchesCondition(entry, cond) {
			return false
		}
	}

	return true
}

// matchesCondition checks if an entry matches a single condition
func matchesCondition(entry interface{}, cond Condition) bool {
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
	field := val.FieldByName(cond.Field)
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
	switch cond.Operator {
	case Equal:
		return fieldStr == cond.Value
	case NotEqual:
		return fieldStr != cond.Value
	case Contains:
		return strings.Contains(fieldStr, cond.Value)
	case GreaterThan, GreaterThanOrEqual, LessThan, LessThanOrEqual:
		// Parse numbers for comparison
		fieldVal, err1 := strconv.ParseFloat(fieldStr, 64)
		condVal, err2 := strconv.ParseFloat(cond.Value, 64)

		if err1 != nil || err2 != nil {
			// Fall back to string comparison if not numeric
			switch cond.Operator {
			case GreaterThan:
				return fieldStr > cond.Value
			case GreaterThanOrEqual:
				return fieldStr >= cond.Value
			case LessThan:
				return fieldStr < cond.Value
			case LessThanOrEqual:
				return fieldStr <= cond.Value
			}
		}

		switch cond.Operator {
		case GreaterThan:
			return fieldVal > condVal
		case GreaterThanOrEqual:
			return fieldVal >= condVal
		case LessThan:
			return fieldVal < condVal
		case LessThanOrEqual:
			return fieldVal <= condVal
		}
	}

	return false
}

// FilterSourcetable filters a sourcetable based on a query
func FilterSourcetable(st ntrip.Sourcetable, queryStr string) (ntrip.Sourcetable, error) {
	query, err := ParseNTRIPQuery(queryStr)
	if err != nil {
		return st, err
	}

	if len(query.Conditions) == 0 {
		return st, nil
	}

	result := ntrip.Sourcetable{
		Casters:  []ntrip.CasterEntry{},
		Networks: []ntrip.NetworkEntry{},
		Mounts:   []ntrip.StreamEntry{},
	}

	// Filter casters
	for _, caster := range st.Casters {
		if query.Matches(caster) {
			result.Casters = append(result.Casters, caster)
		}
	}

	// Filter networks
	for _, network := range st.Networks {
		if query.Matches(network) {
			result.Networks = append(result.Networks, network)
		}
	}

	// Filter mounts
	for _, mount := range st.Mounts {
		if query.Matches(mount) {
			result.Mounts = append(result.Mounts, mount)
		}
	}

	return result, nil
}
