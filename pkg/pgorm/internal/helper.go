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
	switch value.Kind() {
	case reflect.Ptr, reflect.Interface:
		return value.IsNil()
	case reflect.Array, reflect.Slice, reflect.Map, reflect.Chan:
		return value.Len() == 0
	default:
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
