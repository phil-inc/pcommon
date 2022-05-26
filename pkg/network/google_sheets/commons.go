package google_sheets

import (
	"encoding/json"
	"log"
	"time"
)

//LocationPST location America/Los_Angeles
var LocationPST *time.Location

//LocationEST location America/New_York
var LocationEST *time.Location

//ToJSON to JSON string
func ToJSON(data interface{}) string {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		log.Printf("Error marshaling JSON %s\n", err)
		return ""
	}
	return string(jsonBytes)
}
