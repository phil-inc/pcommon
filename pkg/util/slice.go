package util

import "encoding/json"

// StringSliceContains checks if string slice contains given string
func StringSliceContains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// DoesTwoSliceIntersect checks if a string slice contans at least one value of a given slice
func DoesTwoSliceIntersect(s []string, e []string) bool {
	for _, a := range s {
		for _, b := range e {
			if a == b {
				return true
			}
		}
	}
	return false
}

func GetDistinctFromStringSlice(strSlice []string) []string {
	allKeys := make(map[string]bool)
	list := []string{}
	for _, item := range strSlice {
		if _, value := allKeys[item]; !value {
			allKeys[item] = true
			list = append(list, item)
		}
	}
	return list
}

func ConvertStructSliceToMap(structArray interface{}) ([]map[string]interface{}, error) {
	data, err := json.Marshal(structArray) // Convert to a json string
	newMap := make([]map[string]interface{}, 0)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, &newMap) // Convert to a map
	if err != nil {
		return nil, err
	}
	return newMap, nil
}

func IsEqualSliceString(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

func RemoveDuplicateFromSlice(slice []string) []string {
	inResult := make(map[string]bool)
	var result []string
	for _, str := range slice {
		if _, ok := inResult[str]; !ok {
			inResult[str] = true
			result = append(result, str)
		}
	}
	return result
}
