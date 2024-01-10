package util

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/rickar/cal"
)

// NowUTC returns current date time in UTC format
func NowUTC() *time.Time {
	ts := time.Now().UTC()
	return &ts
}

// NowPST returns local pacific time
func NowPST() *time.Time {
	ts := time.Now().In(LocationPST)
	return &ts
}

// NowEST returns est time
func NowEST() *time.Time {
	ts := time.Now().In(LocationEST)
	return &ts
}

// NowInTimeZoneLoc returns time in location
func NowInTimeZoneLoc(loc *time.Location) *time.Time {
	ts := time.Now().In(loc)
	return &ts
}

// YesterdayPST returns local pacific time yesterday
func YesterdayPST() *time.Time {
	now := NowPST()
	t := now.Add(-24 * time.Hour)
	return &t
}

// YesterdayStartEndTimePST start and end time for yesterday
func YesterdayStartEndTimePST() (*time.Time, *time.Time) {
	yest := YesterdayPST()

	ystart := time.Date(yest.Year(), yest.Month(), yest.Day(), 0, 0, 0, 0, yest.Location())
	yend := time.Date(yest.Year(), yest.Month(), yest.Day(), 23, 59, 59, 0, yest.Location())

	return &ystart, &yend
}

// LastDateOfYear return the last day of the year
func LastDateOfYear(year int) time.Time {
	return time.Date(year, 12, 31, 0, 0, 0, 0, LocationPST)
}

// DayStartTimePSTFor returns time based on year, month and day
func DayStartTimePSTFor(year, month, day int) time.Time {
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, LocationPST)
}

// DayEndTimePSTFor returns time based on year, month and day
func DayEndTimePSTFor(year, month, day int) time.Time {
	return time.Date(year, time.Month(month), day, 23, 59, 59, 0, LocationPST)
}

// DayStartTimePST day start time PST
func DayStartTimePST() time.Time {
	//set timezone,
	pst := time.Now().In(LocationPST)
	return time.Date(pst.Year(), pst.Month(), pst.Day(), 0, 0, 0, 0, LocationPST)
}

// DayStartTimeAtGivenHourPST day start time PST
func DayStartTimeAtGivenHourPST(hour int) time.Time {
	//set timezone,
	pst := time.Now().In(LocationPST)
	return time.Date(pst.Year(), pst.Month(), pst.Day(), hour, 0, 0, 0, LocationPST)
}

// DayStartTimeAtGivenDateHourPST day start time  of given date PST
func DayStartTimeAtGivenDateHourPST(givenDate *time.Time, hour int) time.Time {
	//set timezone,
	pst := givenDate.In(LocationPST)
	return time.Date(pst.Year(), pst.Month(), pst.Day(), hour, 0, 0, 0, LocationPST)
}

// DayStartTimeEST day start time EST
func DayStartTimeEST() time.Time {
	//set timezone,
	est := time.Now().In(LocationEST)
	return time.Date(est.Year(), est.Month(), est.Day(), 0, 0, 0, 0, LocationEST)
}

func BusinessHourStartTimeEST() time.Time {
	est := time.Now().In(LocationEST)
	return time.Date(est.Year(), est.Month(), est.Day(), 8, 0, 0, 0, LocationEST)
}

func BusinessHourEndTimeEST() time.Time {
	est := time.Now().In(LocationEST)
	return time.Date(est.Year(), est.Month(), est.Day(), 18, 0, 0, 0, LocationEST)
}

// DayEndTimePST day end time PST
func DayEndTimePST() time.Time {
	pst := time.Now().In(LocationPST)
	return time.Date(pst.Year(), pst.Month(), pst.Day(), 23, 59, 59, 0, LocationPST)
}

// DayStartTimeInTimeZoneLoc day start time in time zone location
func DayStartTimeInTimeZoneLoc(loc *time.Location) time.Time {
	t := time.Now().In(loc)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
}

// DayEndTimeInTimeZoneLoc day end time in time zone location
func DayEndTimeInTimeZoneLoc(loc *time.Location) time.Time {
	t := time.Now().In(loc)
	return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, loc)
}

// DayStartTime returns start date and time of the day
func DayStartTime() time.Time {
	now := time.Now().UTC()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}

// DayEndTime returns end date and time of the day
func DayEndTime() time.Time {
	now := time.Now().UTC()
	return time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, time.UTC)
}

// DateStartTimePST return the start date time for given date as PST
func DateStartTimePST(t *time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, LocationPST)
}

// DateEndTimePST return the end date time for given date as PST
func DateEndTimePST(t *time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, LocationPST)
}

// DateStartTime returns start date and time of specified time
func DateStartTime(t *time.Time, loc *time.Location) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
}

// DayEndTime returns end date and time of specified time
func DateEndTime(t *time.Time, loc *time.Location) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, loc)
}

// MonthStartEndTimePST start and end time for the given year and month in PST
func MonthStartEndTimePST(year, month int) (time.Time, time.Time) {
	monthStart := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, LocationPST)
	monthEnd := time.Date(year, time.Month(month+1), 1, 0, 0, 0, 0, LocationPST)

	return monthStart, monthEnd
}

// MonthStartEndTimePST start and end time for the given year and month in UTC
func MonthStartEndDate(year, month int) (time.Time, time.Time) {
	monthStart := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	monthEnd := monthStart.AddDate(0, 1, -1)

	return monthStart, monthEnd
}

/*
	FortNightStartEndTimePST

returns previous month 2nd fortnight (1, last date) if given date is between 1 -15
else returns 1 -15 of this month
*/
func FortNightStartEndTimePST(year, month, day int) (time.Time, time.Time) {
	var st, et time.Time

	if day < 16 {
		st = time.Date(year, time.Month(month-1), 16, 0, 0, 0, 0, LocationPST)
		et = time.Date(year, time.Month(month), 1, 0, 0, 0, 0, LocationPST)
	} else {
		st = time.Date(year, time.Month(month), 1, 0, 0, 0, 0, LocationPST)
		et = time.Date(year, time.Month(month), 16, 0, 0, 0, 0, LocationPST)
	}

	return st, et
}

// FormatDate formats the date to string using simple format yyyy-mm-dd
func FormatDate(date *time.Time) string {
	return date.Format(localDateFormat)
}

// FormatDateWithTime formats the date to string using simple format yyyy-mm-dd h:mm a tz
func FormatDateWithTime(date *time.Time) string {
	return date.Format(localDateFormatWithTime)
}

// USFormatDate date in US format
func USFormatDate(date *time.Time) string {
	return fmt.Sprintf("%s/%d", date.Format("01/02"), date.Year())
}

// HumanDate returns human readable date
func HumanDate(date *time.Time) string {
	return date.Format("Mon, Jan 2, 2006")
}

// ToPST converts the given UTC date to PST
func ToPST(t *time.Time) *time.Time {
	ts := t.In(LocationPST)
	return &ts
}

// ToEST converts the given UTC date to EST
func ToEST(t *time.Time) *time.Time {
	ts := t.In(LocationEST)
	return &ts
}

// ToTimeZone converts the given UTC date to given time zone
func ToTimeZone(t *time.Time, tz string) *time.Time {
	loc, _ := LoadTimeZoneLocation(tz)
	ts := t.In(loc)
	return &ts
}

// IsSameDate - compares year, month, day and checks to see if they are equal
func IsSameDate(t1, t2 *time.Time) bool {
	if t1 == nil || t2 == nil {
		return false
	}

	return t1.Day() == t2.Day() && t1.Month() == t2.Month() && t1.Year() == t2.Year()
}

// IsDateTodayPST - checks whether the time stamp is today in PST time zone
func IsDateTodayPST(t *time.Time) bool {
	st := DayStartTimePST().UTC()
	et := DayEndTimePST().UTC()
	if t != nil && t.After(st) && t.Before(et) {
		return true
	}
	return false
}

// ConvertUTCToPST converts UTC time to PST
func ConvertUTCToPST(t time.Time) string {
	t = t.In(LocationPST)
	return t.Format(time.RFC822)
}

// IsWeekend checks if the date is in weekend
func IsWeekend(t *time.Time) bool {
	if t.Weekday() == time.Sunday || t.Weekday() == time.Saturday {
		return true
	}

	return false
}

// AddWorkingDays adds working days into the given date
func AddWorkingDays(d time.Time, totalDays int) time.Time {
	for totalDays > 0 {
		if !IsWorkingDay(d) {
			d = GetNextWorkingDay(d)
			continue
		}

		totalDays--
		d = d.Add(time.Hour * 24)
	}

	//check if the last day is itself a holiday
	if !IsWorkingDay(d) {
		d = GetNextWorkingDay(d)
	}

	return d
}

// SubtractWorkingDays subtracts working days into the given date
func SubtractWorkingDays(d time.Time, totalDays int) time.Time {
	for totalDays > 0 {
		if !IsWorkingDay(d) {
			d = GetPreviousWorkingDay(d)
			continue
		}

		totalDays--
		d = d.Add(-1 * time.Hour * 24)
	}

	//check if the last day is itself a holiday
	if !IsWorkingDay(d) {
		d = GetPreviousWorkingDay(d)
	}

	return d
}

// GetPreviousWorkingDay returns the previous working date from the given date
func GetPreviousWorkingDay(d time.Time) time.Time {
	for {
		d = d.Add(-1 * time.Hour * 24)
		if IsWorkingDay(d) {
			return d
		}
	}
}

// GetNextWorkingDay returns the next working date from the given date
func GetNextWorkingDay(d time.Time) time.Time {
	for {
		d = d.Add(time.Hour * 24)
		if IsWorkingDay(d) {
			return d
		}
	}

}

// IsNextDay checks if the given time corresponds to the next day in the PST timezone.
func IsNextDay(d *time.Time) bool {
	if d == nil {
		return false
	}
	nextDay := ToPST(d).AddDate(0, 0, 1).Day()
	return NowPST().Day() == nextDay

}

// GetWorkingDaysBetween calculates the number of working days between two time points, 's' and 'd'.
func GetWorkingDaysBetween(s, d time.Time) int {

	days := 0
	for s.Before(d) {
		if !IsWorkingDay(s) {
			s = GetNextWorkingDay(s)
			continue
		}

		days++
		s = s.Add(time.Hour * 24)
	}
	return days
}

// IsWorkingDay checks if the given time falls on a working day.
// It considers both US holidays and weekends as non-working days.
func IsWorkingDay(d time.Time) bool {
	c := cal.NewCalendar()
	cal.AddUsHolidays(c)
	if c.IsHoliday(d) || IsWeekend(&d) {
		return false
	}
	return true
}

// DaysBetween returns difference between two dates in days.
func DaysBetween(t1, t2 time.Time) int {
	if t1.Before(t2) {
		t1, t2 = t2, t1
	}

	t1 = time.Date(t1.Year(), t1.Month(), t1.Day(), 0, 0, 0, 0, LocationPST)
	t2 = time.Date(t2.Year(), t2.Month(), t2.Day(), 0, 0, 0, 0, LocationPST)
	hours := t1.Sub(t2).Hours()

	return int(hours / 24)
}

// SinceStartOfDayPST returns the start and end times of the current day in the PST timezone.
func SinceStartOfDayPST() (time.Time, time.Time) {

	pst := NowPST()

	st := time.Date(pst.Year(), pst.Month(), pst.Day(), 0, 0, 0, 0, pst.Location())
	et := time.Date(pst.Year(), pst.Month(), pst.Day(), 23, 59, 59, 0, pst.Location())

	return st, et
}

// GetStartOfTheWeek returns the time for start of the current week
func GetStartOfTheWeek(t *time.Time) time.Time {
	daysSinceMonday := StartOfTheWeekMap[t.Weekday()]

	wst := t.Add(time.Duration(-daysSinceMonday*24) * time.Hour)

	wst = time.Date(wst.Year(), wst.Month(), wst.Day(), 0, 0, 0, 0, t.Location())

	return wst.UTC()
}

// SinceStartOfWeekPST calculates the start and end times of the current week in the PST timezone.
func SinceStartOfWeekPST() (time.Time, time.Time) {
	st, et := SinceStartOfDayPST()
	daysSinceMonday := StartOfTheWeekMap[st.Weekday()]

	wst := st.Add(time.Duration(-daysSinceMonday*24) * time.Hour)

	wst = time.Date(wst.Year(), wst.Month(), wst.Day(), 0, 0, 0, 0, st.Location())
	wet := et

	return wst.UTC(), wet.UTC()
}

// SinceStartOfMonthPST calculates the start and end times of the current month in the PST timezone.
// The start time is set to the first day of the month at 00:00:00, and the end time is set to the
// current day at 23:59:59 in PST timezone.
// It returns both times in UTC timezone.
func SinceStartOfMonthPST() (time.Time, time.Time) {
	now := NowPST()

	mst := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	met := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())

	return mst.UTC(), met.UTC()
}

// ParseStringDOB - DOB in string format YYYY-MM-DD
func ParseStringDOB(dob string) *time.Time {
	parts := strings.Split(dob, "-")
	if len(parts) < 3 {
		return nil
	}

	year, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil
	}
	month, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil
	}
	day, err := strconv.Atoi(parts[2])
	if err != nil {
		return nil
	}

	est := time.Date(year, time.Month(month), day, 0, 0, 0, 0, LocationEST)
	utc := est.UTC()

	return &utc
}

// AdjustForHolidays checks if the provided date is a US holiday or a Sunday. If it is, it increments
// the date by one day until a non-holiday, non-Sunday date is found.
// It uses a US holiday calendar to determine holidays.
// The adjusted date is then returned.
func AdjustForHolidays(date time.Time) time.Time {
	c := cal.NewCalendar()
	cal.AddUsHolidays(c)

	startDate := date

	if c.IsHoliday(startDate) || startDate.Weekday() == time.Sunday {
		startDate = startDate.Add(24 * time.Hour)
	}
	//check again just to make sure
	if c.IsHoliday(startDate) || startDate.Weekday() == time.Sunday {
		startDate = startDate.Add(24 * time.Hour)
	}

	return startDate
}

// AdjustForWeekends checks if the provided date is a Saturday or Sunday. If it is, it decrements
// the date by one day until a weekday (Monday to Friday) is found. If the date is before the start
// of the current day, it increments it to the start of the next day.
// The adjusted date is then returned.
func AdjustForWeekends(date time.Time) time.Time {
	if date.Weekday() == time.Sunday {
		date = date.Add(time.Hour * 24 * time.Duration(-1))
	}
	if date.Weekday() == time.Saturday {
		date = date.Add(time.Hour * 24 * time.Duration(-1))
	}
	todayStart := DayStartTimePST()
	if date.Before(todayStart) {
		date = todayStart.AddDate(0, 0, 1)
	}
	return date
}

// ForwardAdjustForWeekends checks if the provided date is a Saturday or Sunday. If it is, it increments
// the date by one day until a weekday (Monday to Friday) is found. If the date is before the start
// of the current day, it increments it to the start of the next day.
// The adjusted date is then returned.
func ForwardAdjustForWeekends(date time.Time) time.Time {
	if date.Weekday() == time.Saturday {
		date = date.Add(time.Hour * 24 * time.Duration(1))
	}
	if date.Weekday() == time.Sunday {
		date = date.Add(time.Hour * 24 * time.Duration(1))
	}
	todayStart := DayStartTimePST()
	if date.Before(todayStart) {
		date = todayStart.AddDate(0, 0, 1)
	}
	return date
}

// RandomDob generates a random dob between 18yr and 40yrs from current date
func RandomDob() *time.Time {
	rdays := rand.Intn(14500-6600) + 6600
	d := NowUTC().AddDate(0, 0, -rdays)
	return &d
}

// GetFormattedLocalTimeByState returns the formatted local time based on state
func GetFormattedLocalTimeByState(state string) string {
	timezone := TimeZoneForState(state)

	location, _ := LoadTimeZoneLocation(timezone)
	localTime := time.Now().In(location)

	formattedDate := localTime.Format("3:04PM") + ", " + fmt.Sprintf("%s/%d", localTime.Format("01/02"), localTime.Year())
	return formattedDate
}

// IsBefore checks if the provided date is before a specified time of day.
func IsBefore(date time.Time, hour, min int) bool {
	hourLocal := time.Date(date.Year(), date.Month(), date.Day(), hour, min, 0, 0, date.Location())
	return date.Before(hourLocal)
}

// IsAfter checks if the provided date is after a specified time of day.
func IsAfter(date time.Time, hour, min int) bool {
	hourLocal := time.Date(date.Year(), date.Month(), date.Day(), hour, min, 0, 0, date.Location())
	return date.After(hourLocal)
}

// IsAfterDate returns true if date1 is after date2 otherwise returns false
func IsAfterDate(date1, date2, format string) bool {
	t1, err := time.Parse(format, date1)
	if err != nil {
		return false
	}
	t2, err := time.Parse(format, date2)
	if err != nil {
		return false
	}
	return t1.After(t2)
}

// IsAfterDate returns true if date1 is before date2 otherwise returns false
func IsBeforeDate(date1, date2, format string) bool {
	t1, err := time.Parse(format, date1)
	if err != nil {
		return false
	}
	t2, err := time.Parse(format, date2)
	if err != nil {
		return false
	}
	return t1.Before(t2)
}

// GetFormattedDateWithoutYearByState formats a given time according to the state's timezone.
// It returns the date in the format "MM/DD" without the year component.
func GetFormattedDateWithoutYearByState(state string, t time.Time) string {
	timezone := TimeZoneForState(state)

	location, _ := LoadTimeZoneLocation(timezone)
	localTime := t.In(location)

	formattedDate := localTime.Format("01/02")
	return formattedDate
}

// YYYYMMDDFormat formats a given time in YYYYMMDD format.
func YYYYMMDDFormat(t *time.Time) string {
	return t.Format(YYYYMMDDFormater)
}

// GetFormattedDateFromString converts a date string in the format "MM/DD/YYYY" or "MM/DD/YY"
// to a *time.Time. If the date string is empty or invalid, it returns nil.
func GetFormattedDateFromString(dateStr string) *time.Time {
	if dateStr == "" {
		return nil
	}

	if len(dateStr) == 10 {
		dob, _ := time.Parse("01/02/2006", dateStr)
		return &dob
	}

	parts := strings.Split(dateStr, "/")
	if len(parts) != 3 {
		return nil
	}

	month, _ := strconv.Atoi(parts[0])
	day, _ := strconv.Atoi(parts[1])
	year, _ := strconv.Atoi(parts[2])

	if year <= 30 && year < 100 {
		year = year + 2000
	} else if year < 100 {
		year = year + 1900
	}

	mm := fmt.Sprintf("%d", month)
	if month < 10 {
		mm = fmt.Sprintf("0%d", month)
	}
	dd := fmt.Sprintf("%d", day)
	if day < 10 {
		dd = fmt.Sprintf("0%d", day)
	}

	dateStr = fmt.Sprintf("%s/%s/%d", mm, dd, year)
	dob, _ := time.Parse("01/02/2006", dateStr)
	return &dob
}

// ToISODateTime converts a given time to its ISO 8601 representation.
func ToISODateTime(t *time.Time) string {
	return t.Format(time.RFC3339)
}

// WithInTimeSpan checks if a given time 'check' falls within the time span defined by 'start' and 'end'.
// It returns true if 'check' is within the span, otherwise it returns false.
func WithInTimeSpan(start, end, check time.Time) bool {
	if (start.Before(check) || start.Equal(check)) && (end.After(check) || end.Equal(check)) {
		return true
	}
	return false
}

// TimeZoneForState returns timezone for given state
func TimeZoneForState(state string) string {
	tz := stateToTimezoneMap[state]
	if tz == "" {
		return PstTimeZone
	}
	return tz
}

// StandardTimeZoneForState returns timezone for given state that maps to one of the 4 standard timezones we support
func StandardTimeZoneForState(state string) string {
	tz := stateToStandardTimeZoneMap[state]
	if tz == "" {
		return CstTimeZone
	}
	return tz
}

// HasMatchingTimeZone returns true if job's trigger timezone matches user timezone.
func HasMatchingTimeZone(ctx context.Context, userState, jobTimeZone string) bool {
	tz := StandardTimeZoneForState(userState)
	if tz == "" {
		tz = CstTimeZone
	}

	return tz == jobTimeZone
}

// LoadTimeZoneLocation loads Location based on given timezone
func LoadTimeZoneLocation(tz string) (*time.Location, error) {
	if timeZoneToLocationMap[tz] == nil {
		loc, err := time.LoadLocation(tz)
		if err != nil {
			return nil, err
		}
		timeZoneToLocationMap[tz] = loc
	}

	return timeZoneToLocationMap[tz], nil
}

// GetRegexFromDateString takes a date string as input and identifies its format using regular expressions.
// It supports various common date formats such as YYYY/MM/DD, MM/DD/YYYY, YYYY-MM-DD, MM-DD-YYYY, YYYY.MM.DD, and MM.DD.YYYY.
// If the input date string does not match any supported format, the function returns an empty string.
func GetRegexFromDateString(ds string) string {
	switch ConvertToRegexPattern(ds) {
	case `^\d\d\d\d/\d\d/\d\d$`:
		return "YYYY/MM/DD"
	case `^\d\d/\d\d/\d\d\d\d$`:
		return "MM/DD/YYYY"
	case `^\d\d\d\d-\d\d-\d\d$`:
		return "YYYY-MM-DD"
	case `^\d\d-\d\d-\d\d\d\d$`:
		return "MM-DD-YYYY"
	case `^\d\d\d\d\.\d\d\.\d\d$`:
		return "YYYY.MM.DD"
	case `^\d\d\.\d\d\.\d\d\d\d$`:
		return "MM.DD.YYYY"
	}

	// Date format not supported
	return ""
}

// GetTimeInStringPST converts the time to PST and formats to local format with time
func GetTimeInStringPST(t time.Time) (string, error) {
	pt, err := ConvertToPST(t)
	if err != nil {
		return "", err
	}

	return pt.Format(localDateFormatWithTime), nil
}

// GetDateInStringPST converts the time to PST and formats to local date format
func GetDateInStringPST(t time.Time) (string, error) {
	pt, err := ConvertToPST(t)
	if err != nil {
		return "", err
	}

	return pt.Format(localDateFormat), nil
}

// ConvertToPST converts the time to PST
func ConvertToPST(t time.Time) (*time.Time, error) {
	loc, err := time.LoadLocation(PstTimeZone)
	if err != nil {
		return nil, err
	}

	pt := t.In(loc)

	return &pt, nil
}

// GetLocDateFromState gets the time based on the given state's timezone
// if state is not found, default is CST TimeZone
// Parameter state should be in abbreviation form. Eg: CA
func GetLocTimeFromState(state string) (*time.Time, error) {
	tzString := StandardTimeZoneForState(state)
	tz, err := LoadTimeZoneLocation(tzString)
	if err != nil {
		return nil, err
	}

	return NowInTimeZoneLoc(tz), nil
}
