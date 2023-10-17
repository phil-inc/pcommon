package util

import (
	"encoding/json"
	"strings"
)

func FlattenMap(input map[string]interface{}) map[string]interface{} {
	flattenedMap := make(map[string]interface{})
	FlattenMapWithPrefix(input, flattenedMap, "")
	return flattenedMap
}

func FlattenMapWithPrefix(input map[string]interface{}, result map[string]interface{}, prefix string) {
	for key, value := range input {
		newKey := key
		if prefix != "" {
			newKey = prefix + "." + key
		}

		switch t := value.(type) {
		case map[string]interface{}:
			FlattenMapWithPrefix(t, result, newKey)
		default:
			result[newKey] = t
		}
	}
}

func ConvertStructToMap(structValue interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(structValue) // Convert to a json string
	newMap := make(map[string]interface{}, 0)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, &newMap) // Convert to a map
	if err != nil {
		return nil, err
	}
	return newMap, nil
}

func ConvertStructToStringMap(structValue interface{}) (map[string]string, error) {
	data, err := json.Marshal(structValue) // Convert to a json string
	newMap := make(map[string]string, 0)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, &newMap) // Convert to a map
	if err != nil {
		return nil, err
	}
	return newMap, nil
}

func GetStringValueFromMap(ikey interface{}, source map[string]interface{}) string {
	if ikey == nil {
		return ""
	}

	key := ikey.(string)
	keys := strings.Split(key, ".")
	val := ""
	holder := source
	lk := len(keys)

	for i, v := range keys {
		if i == (lk - 1) {
			if holder[v] == nil {
				continue
			}
			val = holder[v].(string)
			continue
		}
		holder = source[v].(map[string]interface{})
	}

	return val
}
