package google_sheets

import (
	"fmt"
	"time"
)

//NowPST returns local pacific time
func NowPST() *time.Time {
	ts := time.Now().In(LocationPST)
	return &ts
}

//USFormatDate date in US format
func USFormatDate(date *time.Time) string {
	return fmt.Sprintf("%s/%d", date.Format("01/02"), date.Year())
}
