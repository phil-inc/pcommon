package internal

import (
	"fmt"
	"reflect"
	"strings"
)

func extractColumnsAndValues(model interface{}) ([]string, []interface{}, []string) {
	t := reflect.TypeOf(model)
	v := reflect.ValueOf(model)

	columns := []string{}
	values := []interface{}{}
	placeholders := []string{}

	placeholderCounter := 1

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("db")

		if tag == "" {
			continue
		}

		tagParts := strings.Split(tag, ",")
		columnName := tagParts[0]
		omitEmpty := len(tagParts) > 1 && tagParts[1] == "omitempty"

		fieldValue := v.Field(i)

		if omitEmpty && isZeroValue(fieldValue) {
			continue
		}

		columns = append(columns, columnName)
		values = append(values, fieldValue.Interface())
		placeholders = append(placeholders, fmt.Sprintf("$%d", placeholderCounter))
		placeholderCounter++
	}

	return columns, values, placeholders
}

func isZeroValue(value reflect.Value) bool {
	if !value.IsValid() {
		// Invalid reflect.Value indicates a zero value
		return true
	}

	switch value.Kind() {
	case reflect.Ptr, reflect.Interface:
		if value.IsNil() {
			// A nil pointer or interface is a zero value
			return true
		}
		// If the pointer/interface is non-nil, check the underlying value
		return isZeroValue(value.Elem())
	case reflect.Array:
		// Check if all elements in the array are zero
		for i := 0; i < value.Len(); i++ {
			if !isZeroValue(value.Index(i)) {
				return false
			}
		}
		return true
	case reflect.Slice, reflect.Map, reflect.Chan:
		return value.IsNil() || value.Len() == 0
	case reflect.Struct:
		if value.Type().String() == "time.Time" {
			// Special handling for time.Time zero value
			return value.Interface() == reflect.Zero(value.Type()).Interface()
		}
		// Check if all fields in the struct are zero
		for i := 0; i < value.NumField(); i++ {
			if !isZeroValue(value.Field(i)) {
				return false
			}
		}
		return true
	default:
		// Compare with the zero value of the same type
		zeroValue := reflect.Zero(value.Type())
		return reflect.DeepEqual(value.Interface(), zeroValue.Interface())
	}
}

func replacePlaceholders(condition string, startIndex int) string {
	result := ""
	placeholderCount := startIndex
	for _, char := range condition {
		if char == '?' {
			placeholderCount++
			result += fmt.Sprintf("$%d", placeholderCount)
		} else {
			result += string(char)
		}
	}
	return result
}
