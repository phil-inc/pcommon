package util

import (
	"database/sql"
	"fmt"
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

// Formats postgres returned bool value to Yes/No or "Y,N"
func FormatSQLNullBool(b sql.NullBool, format string) string {
	f := strings.Split(format, ",")

	if !b.Valid {
		return ""
	}

	if b.Bool {
		return f[0]
	}

	return f[1]

}

func FormatSQLNullFloat(f sql.NullFloat64) string {

	if !f.Valid {
		return ""
	}

	v := f.Float64
	s := fmt.Sprintf("%.2f", v)
	return strings.TrimRight(strings.TrimRight(s, "0"), ".")
}

func FormatSQLNullInt(i sql.NullInt64) string {

	if !i.Valid {
		return ""
	}

	v := i.Int64
	return fmt.Sprint(v)
}

func FormatSQLNULLDateWithTimePST(date sql.NullTime, format string) string {

	if !date.Valid {
		return ""
	}

	dt := date.Time

	if format != "" {
		return ToPST(&dt).Format(format)
	}

	return ToPST(&dt).Format(localDateFormatWithTime)
}

func FormatSQLNULLDateWithTimeEST(date sql.NullTime, format string) string {

	if !date.Valid {
		return ""
	}

	dt := date.Time

	if format != "" {
		return ToEST(&dt).Format(format)
	}

	return ToEST(&dt).Format(localDateFormatWithTime)
}

func FormatSQLNullString(s sql.NullString) string {

	if !s.Valid {
		return "-"
	}
	return s.String
}
