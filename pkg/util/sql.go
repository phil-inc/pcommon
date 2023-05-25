package util

import (
	"database/sql"
	"strings"
)

// Formats postgres returned date to specified date format
func FormatSQLNullDate(t sql.NullTime, format string) string {

	if !t.Valid {
		return ""
	}

	f := strings.Split(format, ",")

	date := t.Time

	return ToPST(&date).Format(f[0])
}

func FormatSQLNullString(s sql.NullString) string {

	if !s.Valid {
		return "-"
	}
	return s.String
}
