package util

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"regexp"
	"strconv"
	"time"
	"unicode"

	"strings"

	"log"

	"reflect"

	"github.com/narup/gconfig"
	logger "github.com/phil-inc/plog-ng/pkg/core"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

///// Common functions ///////

func init() {
	rand.Seed(time.Now().UnixNano())

	ploc, err := time.LoadLocation(PstTimeZone)
	if err != nil {
		log.Panicf("ERROR loading location")
		return
	}
	LocationPST = ploc

	eloc, err := time.LoadLocation(EstTimeZone)
	if err != nil {
		log.Panicf("ERROR loading location")
		return
	}
	LocationEST = eloc
}

func IsWinterMonth() bool {
	cm := int(time.Now().Month())
	if cm == 12 || cm == 1 || cm == 2 || cm == 3 {
		return true
	}
	return false
}

// Index returns index for a given string in an array of  string
func Index(vs []string, t string) int {
	for i, v := range vs {
		if v == t {
			return i
		}
	}
	return -1
}

// Include check if string is in an array
func Include(vs []string, t string) bool {
	return Index(vs, t) >= 0
}

// HandleRefLoadError wraps all the given errors
func HandleRefLoadError(message string, errs []error) error {
	var err error
	for i := range errs {
		if errs[i] != nil {
			if err == nil {
				err = errs[i]
			}
			err = errors.Wrapf(err, message)
		}
	}
	return err
}

// WrapErrors wraps all the given errors
func WrapErrors(message string, errs []error) error {
	var err error
	for i := range errs {
		if errs[i] != nil {
			if err == nil {
				err = errs[i]
			}
			err = errors.Wrapf(err, message)
		}
	}
	return err
}

// Config returns string configuration for given key
func Config(key string) string {
	//TODO shall I feature flag it?
	//return strings.ReplaceAll(gconfig.Gcg.GetString(key), `\n`, "\n")
	//return strings.ReplaceAll(gconfig.Gcg.GetStringOrDefault(key), `\n`, "\n")
	return strings.ReplaceAll(gconfig.Gcg.GetStringOrDefaultInCommaSeparator(key), `\n`, "\n")

}

// BoolConfig returns config boolean value for the given key
func BoolConfig(key string) bool {
	return gconfig.Gcg.GetBool(key)
}

// IntConfig returns config integer value for the given key
func IntConfig(key string) int {
	return gconfig.Gcg.GetInt(key)
}

// SafeConfig returns empty string if the key doesn't exists
func SafeConfig(key string) string {
	if gconfig.Gcg.Exists(key) {
		return gconfig.Gcg.GetString(key)
	}
	return ""
}

func GetString(s string) string {
	if s == "" {
		return "N/A"
	}
	return s
}

// RandomToken generates random string token
func RandomToken(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = alphanumericRunes[rand.Intn(len(alphanumericRunes))]
	}
	return string(b)
}

// RandomAlphabeticToken generates random string token comprised of only letters
func RandomAlphabeticToken(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = alphabeticRunes[rand.Intn(len(alphabeticRunes))]
	}
	return string(b)
}

// IsPhoneNumber returns true if the given string matches phone number format. It's a best effort result no guarantee
func IsPhoneNumber(str string) bool {
	re := regexp.MustCompile(`^\(*\+*[1-9]{0,3}\)*-*[1-9]{0,3}[-. /]*\(*[2-9]\d{2}\)*[-. /]*\d{3}[-. /]*\d{4} *e*x*t*\.* *\d{0,4}$`)
	return re.MatchString(str)
}

// FormatPhoneToZendeskFormat formats the given phone number to: +1XXXXXXXXXX
func FormatPhoneToZendeskFormat(phoneNumber string) string {
	return FormatNumber(phoneNumber, true)
}

// FormatPhone formats the given phone number
func FormatPhone(phoneNumber string) string {
	return FormatNumber(phoneNumber, false)
}

func FormatNumber(phoneNumber string, zendeskFormat bool) string {
	if phoneNumber == "" {
		return ""
	}
	phoneNumber = SanitizePhoneNumber(phoneNumber)
	if len(phoneNumber) < 10 {
		return phoneNumber
	}
	if zendeskFormat {
		return fmt.Sprintf("+1%s", phoneNumber[:10])
	}
	return fmt.Sprintf("(%s) %s-%s", phoneNumber[:3], phoneNumber[3:6], phoneNumber[6:10])
}

// FormatValidPhone formats the given phone number only if it's a valid phone number
func FormatValidPhone(phoneNumber string) string {
	if IsPhoneNumber(phoneNumber) {
		return FormatPhone(phoneNumber)
	}

	return phoneNumber
}

// SanitizePhoneNumber cleans up the phone number
func SanitizePhoneNumber(phoneNumber string) string {
	if strings.HasPrefix(phoneNumber, "1.") {
		phoneNumber = strings.Replace(phoneNumber, "1.", "", -1)
	}

	phoneNumber = strings.Replace(phoneNumber, "(", "", -1)
	phoneNumber = strings.Replace(phoneNumber, ")", "", -1)
	phoneNumber = strings.Replace(phoneNumber, " ", "", -1)
	phoneNumber = strings.Replace(phoneNumber, "-", "", -1)
	phoneNumber = strings.Replace(phoneNumber, ".", "", -1)

	if strings.HasPrefix(phoneNumber, "+1") {
		phoneNumber = strings.Replace(phoneNumber, "+1", "", -1)
	}

	// Return empty if all strings are zero
	if strings.Trim(phoneNumber, "0") == "" {
		return ""
	}

	return phoneNumber
}

// FormatOrderNumber formats the order number to xxxx-xxxx-xxxx
func FormatOrderNumber(orderNumber string) string {
	orderNumber = SanitizeOrderNumber(orderNumber)
	if len(orderNumber) < 12 {
		return orderNumber
	}
	return fmt.Sprintf("%s-%s-%s", orderNumber[:4], orderNumber[4:8], orderNumber[8:12])
}

// SanitizeOrderNumber cleans up the order number
func SanitizeOrderNumber(orderNumber string) string {
	orderNumber = strings.Replace(orderNumber, "-", "", -1)
	return orderNumber
}

// SanitizeZip cleans up the zipcode
func SanitizeZip(zipCode string) string {
	zipCode = strings.TrimSpace(zipCode)
	if len(zipCode) < 5 {
		return fmt.Sprintf("%05s", zipCode)
	}
	if len(zipCode) > 5 {
		return zipCode[:5]
	}
	return zipCode
}

// SanitizeStreetAddress cleans up street address
func SanitizeStreetAddress(address string) string {
	address = strings.Replace(address, "(", "", -1)
	address = strings.Replace(address, ")", "", -1)
	address = strings.Replace(address, ".", "", -1)
	address = strings.Replace(address, ",", "", -1)

	// replace all multiple whitespaces to single
	re := regexp.MustCompile(`\s+`)
	address = re.ReplaceAllString(address, " ")

	address = strings.Trim(address, " ")

	return strings.ToUpper(address)
}

// AddStringValues adds 2 string formatted float numbers
func AddStringValues(v1, v2 string) string {
	val1, err := strconv.ParseFloat(v1, 64)
	if err != nil {
		val1 = 0.0
	}

	val2, err := strconv.ParseFloat(v2, 64)
	if err != nil {
		val2 = 0.0
	}

	result := val1 + val2
	return fmt.Sprintf("%.2f", result)
}

// SubtractStringValues subtracts v2 by v1
func SubtractStringValues(v1, v2 string) string {
	val1, err := strconv.ParseFloat(v1, 64)
	if err != nil {
		val1 = 0.0
	}

	val2, err := strconv.ParseFloat(v2, 64)
	if err != nil {
		val2 = 0.0
	}

	result := val1 - val2
	return fmt.Sprintf("%.2f", result)
}

// FormatPriceForDisplay formats the price in string format as 2 decimal value
func FormatPriceForDisplay(val interface{}) string {
	fval := correctFloatValue(val)
	return fmt.Sprintf("%.2f", fval)
}

// FormatPriceForDB formats the given currency as 2 decimal place USD only if it's decimal
func FormatPriceForDB(val interface{}) string {
	fval := correctFloatValue(val)
	price := fmt.Sprintf("%.2f", fval)
	if strings.HasSuffix(price, ".00") {
		price = strings.Replace(price, ".00", "", -1)
	}

	return price
}

// USDFormat formats the given currency as 2 decimal place USD
func USDFormat(val interface{}) string {
	fval := correctFloatValue(val)
	if fval < 0 {
		fval = math.Abs(fval)
		return fmt.Sprintf("-$%.2f", fval)
	}
	return fmt.Sprintf("$%.2f", fval)
}

// TruncatedUSDFormat formats the given currency value to remove trailing ".00" if present
func TruncatedUSDFormat(val interface{}) string {
	formattedCurrency := USDFormat(val)
	if strings.HasSuffix(formattedCurrency, ".00") {
		return strings.Replace(formattedCurrency, ".00", "", -1)
	}
	return formattedCurrency
}

// RemoveUSDFormat
func RemoveUSDFormat(val interface{}) string {
	fval := correctFloatValue(val)
	return fmt.Sprintf("%.2f", fval)
}

// FloatValue returns float64 value for a passed in value
func FloatValue(val interface{}) float64 {
	return correctFloatValue(val)
}

// IsLocal checks if it's local environment
func IsLocal() bool {
	return Config("app.environment") == "local" || Config("app.environment") == "local-dev"
}

// IsPureLocal checks if it's local environment
func IsPureLocal() bool {
	return Config("app.environment") == "local"
}

// IsRemotePublishEventsDisabled checks if publishing to remote events is enabled
func IsRemotePublishEventsDisabled() bool {
	// always disable in feature envs since it can be running an old version of capi
	return IsFeatureEnvironment() || Config("event.remote.publish.disabled") == "true"
}

// IsRemotePublishEventsDisabled checks if publishing to remote events is enabled
func IsRemoteBroadcastEventsDisabled() bool {
	// always disable in feature envs since it can be running an old version of capi
	return IsFeatureEnvironment() || Config("event.remote.broadcast.disabled") == "true"
}

// IsDev returns if the application is running in dev environment
func IsDev() bool {
	return Config("app.environment") == "dev" || Config("app.environment") == "local" || Config("app.environment") == "local-dev"
}

// IsProd check if application is running in prod env
func IsProd() bool {
	return Config("app.environment") == "prod"
}

// IsStage check if application is running in stage env
func IsStage() bool {
	return Config("app.environment") == "stage"
}

// IsDebugMode check if app is running in debug mode. Does heavy logging
func IsDebugMode() bool {
	return gconfig.Gcg.GetBool("app.debugMode")
}

// IsFeatureEnvironment check if the app is running inside a feature env
func IsFeatureEnvironment() bool {
	// feature envs have override urls set
	return os.Getenv("IS_FEATURE_ENVIRONMENT") != ""
}

// IsRunningOnMinimalSeedDB check if the env is using minimal seed db (within feature env)
func IsRunningOnMinimalSeedDB() bool {
	return os.Getenv("HAS_MINIMAL_SEED_DB") != ""
}

// ShouldStartOrderExport check if the order export is enabled
func IsOrderExportEnabled() bool {
	// feature envs can be running an old version of capi, which might export wrong data
	// so if the app is running inside feature env, disable unless it's using a minimal seed db
	if IsFeatureEnvironment() {
		return IsRunningOnMinimalSeedDB()
	}

	// enable for dev and prod
	return (IsDev() && !IsLocal()) || IsProd()
}

// IsReadOnlyMode check if application is running in read only mode
func IsReadOnlyMode() bool {
	return Config("app.environment") == "read-only"
}

// IsClosed checks if passed in channel is closed
func IsClosed(ch <-chan string) bool {
	select {
	case <-ch:
		return true
	default:
	}

	return false
}

// ToJSON to JSON string
func ToJSON(data interface{}) string {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		logger.Errorf("Error marshaling JSON %s", err)
		return ""
	}
	return string(jsonBytes)
}

// TypeName returns the string value of the interface v
func TypeName(v interface{}) string {
	return reflect.TypeOf(v).String()
}

// IsSuffix check if the name is a suffix
func IsSuffix(name string) bool {
	return nameSuffixes[name]
}

// StripSuffix Strip suffix returns an array of the input name separated by spaces and stripped of suffix
func StripSuffix(fullName string) []string {
	r, _ := regexp.Compile(`[^a-zA-Z0-9\pL  \x60\-\_\^\~\'·\/]+`)
	//r, _ := regexp.Compile(`[^ \00c0 - \01ff \pL a-zA-Z'\-]+$`)
	alphaNumericName := r.ReplaceAllString(strings.ToLower(fullName), "")
	fullNameFields := strings.Fields(alphaNumericName)

	//loop through name and throw out suffixes
	var fullNameWithoutSuffix []string
	for _, field := range fullNameFields {
		if IsSuffix(field) {
			continue
		}

		fullNameWithoutSuffix = append(fullNameWithoutSuffix, field)
	}

	return fullNameWithoutSuffix
}

// FirstName returns the first name only for the given full name
func FirstName(fullName string) string {
	s := strings.Split(fullName, " ")
	if len(s) > 0 {
		fn := strings.ToLower(s[0])
		return TitleCase(fn)
	}
	return fullName
}

// MiddleName returns the middle name only for the given full name
func MiddleName(fullName string) string {
	s := strings.Split(fullName, " ")
	if len(s) > 2 {
		mn := strings.Join(s[1:len(s)-1], " ")
		mn = strings.ToLower(mn)
		return TitleCase(mn)
	}
	return ""
}

// LastName returns the last name only for the given full name
func LastName(fullName string) string {
	s := StripSuffix(fullName)
	// s := strings.Split(fullName, " ")
	if len(s) > 0 {
		ln := strings.ToLower(s[len(s)-1])
		return TitleCase(ln)
	}
	return fullName
}

// PartialName returns the full first name and first letter of the middle or last name
func PartialName(fullName string) string {
	s := strings.Split(fullName, " ")
	if len(s) > 0 {
		fn := strings.ToLower(s[0])
		ln := strings.ToLower(s[len(s)-1])

		firstChar := string(ln[0])
		pn := fmt.Sprintf("%s %s", fn, firstChar)

		return TitleCase(pn)
	}
	return fullName
}

// IsMatchingLastName returns whether or not the last name provided
// matches the last name of the full name provided
func IsMatchingLastName(fullName string, lastName string) bool {
	// we must handle cases where last name is multiple words
	if fullName == "" {
		return false
	}

	if lastName == "" {
		return false
	}

	// if fullName is only one word, return false
	firstSpaceIdx := strings.Index(fullName, " ")
	if firstSpaceIdx == -1 {
		return false
	}

	// last name must not include the first word in fullName
	maxLastNameLength := len(fullName) - firstSpaceIdx
	if len(lastName) > maxLastNameLength {
		return false
	}

	minLastName := LastName(fullName)
	if len(lastName) < len(minLastName) {
		return false
	}

	fullNameNoSpace := strings.Join(StripSuffix(fullName), "")
	lastNameNoSpace := strings.ToLower(strings.Replace(lastName, " ", "", -1))

	lenLastName := len(lastNameNoSpace)
	lenFullName := len(fullNameNoSpace)

	lastNameStartIdx := lenFullName - lenLastName
	if len(fullNameNoSpace) < lastNameStartIdx || lastNameStartIdx < 0 {
		return false
	}

	if fullNameNoSpace[lastNameStartIdx:] != lastNameNoSpace {
		return false
	}

	return true
}

// TrimAndLower trim the input string and also convert to lower case
func TrimAndLower(input string) string {
	t := strings.TrimSpace(input)
	return strings.ToLower(t)
}

// TrimAndUpper trim the input string and also convert to upper case
func TrimAndUpper(input string) string {
	t := strings.TrimSpace(input)
	return strings.ToUpper(t)
}

// TrimAndTitle trim the input string and also convert to title case
func TrimAndTitle(input string) string {
	t := TrimAndLower(input)
	return TitleCase(t)
}

// Trim trim the input string
func Trim(input string) string {
	return strings.TrimSpace(input)
}

func SameStringIgnoreCase(str1, str2 string) bool {
	return TrimAndLower(str1) == TrimAndLower(str2)
}

// GetFirstWordsFromString returns the value of the first (count) number of words from a given string
func GetFirstWordsFromString(value string, count int) string {
	for i := range value {
		if value[i] == ' ' {
			count--
			if count == 0 {
				return value[0:i]
			}
		}
	}
	return value
}

// CleanUpEmail Clean up an email we receive in an API request
func CleanUpEmail(email string) string {
	te := TrimAndLower(email)
	if strings.HasSuffix(te, ".con") {
		te = strings.Replace(te, ".con", ".com", 1)
	}
	return te
}

// Check if email is valid or not
func IsEmailValid(email string) bool {
	// Regular expression pattern for email validation
	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`

	regex := regexp.MustCompile(pattern)

	return regex.MatchString(email)
}

// RemoveSpaces removes all the spaces of a string
func RemoveSpaces(value string) string {
	return strings.Replace(value, " ", "", -1)
}

// CleanDoctorName clean up the doctor name to remove prefix and suffixes
func CleanDoctorName(name string) string {
	name = strings.ToLower(name)
	if strings.HasPrefix(name, "dr.") {
		name = strings.Replace(name, "dr.", "", -1)
	}
	if strings.HasSuffix(name, " md") {
		name = strings.Replace(name, " md", "", -1)
	}
	if strings.HasSuffix(name, " m.d.") {
		name = strings.Replace(name, " m.d.", "", -1)
	}
	if strings.HasSuffix(name, " pa") {
		name = strings.Replace(name, " pa", "", -1)
	}
	if strings.HasSuffix(name, " p.a.") {
		name = strings.Replace(name, " p.a.", "", -1)
	}
	if strings.HasSuffix(name, " n.p.") {
		name = strings.Replace(name, " n.p.", "", -1)
	}

	name = TrimAndTitle(name)

	return name
}

func GetFullStateNameFromAbbreviation(state string) string {
	return StateAbbreviation[state]
}

// Get state abbreviation from full name
func GetStateAbbreviationFromName(state string) string {
	if len(state) <= 2 {
		return state
	}

	for k, v := range StateAbbreviation {
		if strings.EqualFold(v, state) {
			return k
		}
	}

	return state
}

func correctFloatValue(val interface{}) float64 {
	var fval float64

	switch price := val.(type) {
	case string:
		price = strings.Replace(price, "$", "", -1)
		if price == "" || price == "0.0" || price == "0" || price == "0.00" {
			fval = 0.
		}
		f, err := strconv.ParseFloat(price, 64)
		if err != nil {
			fval = 0.0
		}
		fval = f
	case float64:
		fval = price
	case int:
		fval = float64(price)
	case float32:
		fval = float64(price)
	}

	return fval
}

func AddUtmSource(authURL string, src string) string {
	return fmt.Sprintf("%s?%s=%s", authURL, UtmSource, src)
}

// Remove sensitive information
func GetDiscreetRxName(rxName string) string {
	if rxName == "" {
		logger.Warn("Rx name cannot be empty.")
	}

	rxName = strings.ToUpper(rxName)
	if len(rxName) < 3 { //edge case, when rx name is less than 3 characters
		return strings.TrimSpace(rxName) + "••••"
	}

	return strings.TrimSpace(rxName[0:3]) + "••••"
}

func IsInternalEmail(email string) bool {
	return strings.Contains(email, "phil.us") || strings.Contains(email, "usephil.com")
}

// StringData fetch string data from events data map
func StringData(d interface{}) string {
	if d != nil {
		switch v := d.(type) {
		case string:
			return v
		default:
			return ""
		}
	}
	return ""
}

// GetInteger return the integer part of a number
func GetInteger(str string) string {
	number, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return ""
	}

	return strconv.Itoa(int(number))
}

// GenerateRandomBytes returns securely generated random bytes.
// It will return an error if the system's secure random
// number generator fails to function correctly, in which
// case the caller should not continue.
func GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	// Note that err == nil only if we read len(b) bytes.
	if err != nil {
		return nil, err
	}

	return b, nil
}

// GenerateRandomString returns a URL-safe, base64 encoded
// securely generated random string.
// It will return an error if the system's secure random
// number generator fails to function correctly, in which
// case the caller should not continue.
func GenerateRandomString(s int) (string, error) {
	b, err := GenerateRandomBytes(s)
	return base64.URLEncoding.EncodeToString(b), err
}

func GetTempEmail(ptName, ptPhoneNumber string) string {
	//temp email
	tempPtName := strings.Replace(ptName, " ", "", -1)

	ts := strconv.FormatInt(time.Now().Unix(), 10)
	tempEmail := fmt.Sprintf("%s+%s%s-temp@phil.us", tempPtName, ptPhoneNumber, ts)
	return strings.ToLower(tempEmail)
}

// StringArrayContains compares the value in dataset
func StringArrayContains(dataSets []string, str string) bool {
	for _, value := range dataSets {
		if value == str {
			return true
		}
	}

	return false
}

func PadBin(s string) string {
	l := len(s)
	if l >= 6 {
		return s
	}

	padding := strings.Repeat("0", 6-l)
	return padding + s
}

// To check whether a string is included in the array or not.
func Contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// GetMaskedEmail replace middle charaters of email address with  3 asterik(***)
func GetMaskedEmail(email string) string {
	if email == "" {
		return ""
	}

	strArr := strings.Split(email, "@")

	if len(strArr) < 2 {
		return ""
	}

	username := strArr[0]
	usernameLength := len(username)

	domain := strArr[1]
	n := strings.Index(domain, ".")

	if n == -1 {
		return ""
	}

	return fmt.Sprintf("%s***%s@%s***%s%s", username[0:1], username[usernameLength-1:], domain[0:1], domain[n-1:n], domain[n:])
}

// GetMaskedPhone returns last 4 digits of phone number
func GetMaskedPhone(phone string) string {
	if len(phone) <= 4 {
		return phone
	}

	return phone[len(phone)-4:]
}

func ExtractDate(date string) (*time.Time, error) {
	d, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, err
	}
	t := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, LocationPST).UTC()
	return &t, nil
}

func OnlyDigits(s string) bool {
	sampleRegexp := regexp.MustCompile(`^\d+$`)
	return sampleRegexp.MatchString(s)
}

func CheckDigitString(s string, n int) bool {
	exp := fmt.Sprintf(`\d{%v}`, n)
	sampleRegexp := regexp.MustCompile(exp)
	return sampleRegexp.MatchString(s)
}

func IsMatchingEmail(a string, b string) bool {
	return strings.EqualFold(a, b)
}

// valid two factor code includes
// xxxxxx (6-digit code)
func CleanTwoFactorCode(authCode string) string {
	if len(authCode) > 6 {
		return authCode[:6]
	}

	return authCode
}

func PadPCN(s string) string {
	l := len(s)
	if l >= 10 {
		return s
	}
	padding := strings.Repeat(" ", 10-l)
	return s + padding
}
func PCNValidation(s string) (string, error) {
	l := len(s)
	if l > 10 {
		return s, errors.New("Invalid PCN")
	}
	return PadPCN(s), nil
}

func BinValidation(s string) (string, error) {
	l := len(s)
	if l == 0 || l > 6 {
		return s, errors.New("Invalid BIN")
	}
	return PadBin(s), nil
}

func ContainsInsensitive(a string, b ...string) bool {
	for _, sub := range b {
		if strings.Contains(strings.ToLower(a), strings.ToLower(sub)) {
			return true
		}
	}
	return false
}

// ToFixedPrecision Rounds the float to nearest like 12.3456 -> 12.35
func ToFixedPrecision(val float64, precision int) float64 {
	return math.Round(val*(math.Pow10(precision))) / math.Pow10(precision)
}

func CommonLogger(s string, payload interface{}) {
	if IsDev() {
		if payload != nil {
			v, e := json.Marshal(payload)
			if e != nil {
				logger.Infof(s + " phil:: Unable to parse the data")
			} else {
				logger.Infof(s+" payload Truepill:: %s", string(v))
			}
		} else {
			logger.Info(s)
		}

	}
}

func AddLeadingZero(part string) string {
	return "0" + part
}

func ConvertSecondsToHours(v float64) string {
	if v == 0 {
		return ""
	}
	return FormatFloat(v / (60 * 60))
}

func ConvertSecondsToDays(v float64) int {
	if v == 0 {
		return 0
	}

	return ConvertFormatToInt((v / (60 * 60)) / 24)
}

func ConvertFormatToInt(v float64) int {
	s := fmt.Sprintf("%.0f", v)
	if i, err := strconv.Atoi(s); err == nil {
		return i
	} else {
		return 0
	}
}

func FormatFloat(v float64) string {
	s := fmt.Sprintf("%.3f", v)
	return strings.TrimRight(strings.TrimRight(s, "0"), ".")
}

// IsInArray finds if given string is in array
func IsInArray(s string, all []string) bool {
	if s == "" {
		return false
	}
	s = strings.ToLower(s)
	for _, str := range all {
		if str != "" && s == strings.ToLower(str) {
			return true
		}
	}
	return false
}

func ContentPartialFilePath(fileName string) string {
	if fileName == "" {
		return ""
	}
	return fmt.Sprintf("templates/partial-templates/%s", fileName)
}

func GetPhilLogoURL() string {
	return fmt.Sprintf("%s/img/insert-card/p-new-log-black.svg", Config("dashboard.server.url"))
}

func IsGreaterThan(number, value float64) bool {
	return number > value
}

// Removes empty string from string slice. ["abc", " ", "", "bcd"] --> ["abc" "bcd"]
func TrimSlice(ss []string) []string {
	var rs []string
	for _, s := range ss {
		if strings.Trim(s, " ") != "" {
			rs = append(rs, s)
		}
	}

	return rs
}

// MergeStringSlices merges 'a' and 'b', excluding duplicates from 'b' already present in 'a'.
func MergeStringSlices(a, b []string) []string {
	seen := make(map[string]bool)

	for _, v := range a {
		seen[v] = true
	}

	for _, v := range b {
		if !seen[v] {
			a = append(a, v)
			seen[v] = true
		}
	}

	return a
}

// matches following date format
// YYYY-MM-DD
// MM-DD-YYYY
// YYYY/MM/DD
// MM/DD/YYYY
// YYYY.MM.DD
// MM.DD.YYYY
func IsDateString(inputDate string) bool {
	dateFormats := []string{
		`^\d{4}/\d{2}/\d{2}$`,
		`^\d{2}/\d{2}/\d{4}$`,
		`^\d{4}-\d{2}-\d{2}$`,
		`^\d{2}-\d{2}-\d{4}$`,
		`^\d{4}\.\d{2}\.\d{2}$`,
		`^\d{2}\.\d{2}\.\d{4}$`,
	}

	for _, format := range dateFormats {
		re := regexp.MustCompile(format)
		if re.MatchString(inputDate) {
			return true
		}
	}

	return false
}

// Gives regex pattern
// YYYY-MM-DD will give ^\d\d\d\d-\d\d-\d\d$
func ConvertToRegexPattern(input string) string {
	var builder strings.Builder

	for _, ch := range input {
		switch ch {
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			builder.WriteString(`\d`)
		default:
			builder.WriteString(regexp.QuoteMeta(string(ch)))
		}
	}

	return fmt.Sprintf("^%s$", builder.String())
}

// Converts every first letter of the  word in the sentence to uppercase
func TitleCase(input string) string {
	return cases.Title(language.AmericanEnglish).String(input)
}

// SnakeCase converts the given string to snake case
func SnakeCase(str string) string {
	matchFirstCap := regexp.MustCompile("(.)([A-Z][a-z]+)")
	matchAllCap := regexp.MustCompile("([a-z0-9])([A-Z])")

	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

// CastValue cast the given value into the given type
func CastValue(val interface{}, valueType string) interface{} {
	switch valueType {
	case "string":
		return cast.ToString(val)
	case "int":
		return cast.ToInt(val)
	case "bool":
		return cast.ToBool(val)
	case "float64":
		fallthrough
	case "float":
		return cast.ToFloat64(val)
	case "time":
		return cast.ToTime(val)
	case "StringArray":
		var res []string
		err := json.Unmarshal([]byte(val.(string)), &res)
		if err != nil {
			logger.Errorf("Error parsing value %+v. Error %s", val, err.Error())
			return val
		}
		return res
	case "IntArray":
		return cast.ToIntSlice(val)
	case "StringMap":
		return cast.ToStringMap(val)
	case "StringMapString":
		return cast.ToStringMapString(val)
	}
	return val
}

// YesNo change and return boolean to yes or no
func YesNo(b bool) string {
	if b {
		return "Yes"
	}
	return "No"
}

// IsAlphanumeric checks if the string is alphanumeric
func IsAlphanumeric(s string) bool {
	return regexp.MustCompile(`^[a-zA-Z0-9]*$`).MatchString(s)
}

/*
IsBinValid validates bin
The criterias checked are:
1. If the string contains all numeric values
2. The length of the string is six.
*/
func IsBinValid(s string) bool {
	// bin should only contain numeric values
	if _, err := strconv.Atoi(s); err != nil {
		return false
	}

	// bin should be 6 digits
	return len(s) == 6
}

// IsInsuranceIDValid check if the insurance ID is valid
func IsInsuranceIDValid(s string) bool {
	return IsAlphanumeric(s)
}

// IsPCNValid check if the pcn is valid
func IsPCNValid(s string) bool {
	return IsAlphanumeric(s)
}

// GetMaskedName replace last name with *****
func GetMaskedName(fullName string) string {
	return fmt.Sprintf("%s %s", FirstName(fullName), "*****")
}

// IsValidPassword validates password to have at least 8 character with number and alphabets
func IsValidPassword(password string) bool {
	var numberPresent bool
	var letterPresent bool
	const minPassLength = 8

	for _, ch := range password {
		switch {
		case unicode.IsNumber(ch):
			numberPresent = true
		case unicode.IsLetter(ch):
			letterPresent = true
		}
	}

	if len(password) >= minPassLength && numberPresent && letterPresent {
		return true
	}
	return false
}

// Cleans the input string by replacing all characters except alphanumeric characters, spaces, slashes, underscores, dashes, caret, @, plus, and dot with space, and remove multiple white spaces
func CleanSearchText(searchText string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9\s/_\-\^\.\@\+]`)
	st := re.ReplaceAllString(searchText, " ")
	return NormalizeSpace(st)
}

// Replace multiple spaces with a single space
func NormalizeSpace(s string) string {
	space := regexp.MustCompile(`\s+`)
	return space.ReplaceAllString(s, " ")
}

// ArePointerValuesEqual compares two pointers of any type, including *time.Time pointers.
// If both pointers are nil, it returns true. If only one is nil, it returns false.
// For *time.Time pointers, it uses the time.Time.Equal method for accurate comparison.
// For other pointer types, it uses reflect.DeepEqual to compare the dereferenced values.
//
// Parameters:
//   - p1: The first pointer to compare.
//   - p2: The second pointer to compare.
//
// Returns:
//   - A boolean value indicating whether the two pointers are equal.
func ArePointerValuesEqual(p1, p2 interface{}) bool {
	// If either pointer is nil, return whether they are both nil
	if p1 == nil || p2 == nil {
		return p1 == p2
	}

	// Handle *time.Time comparison explicitly
	if t1, ok1 := p1.(*time.Time); ok1 {
		if t2, ok2 := p2.(*time.Time); ok2 {
			return t1.Equal(*t2)
		}
	}

	v1 := reflect.ValueOf(p1)
	v2 := reflect.ValueOf(p2)

	// Check if either pointer is nil after the interface checks
	if v1.IsNil() || v2.IsNil() {
		// Return false if one is nil and the other is not
		return v1.IsNil() == v2.IsNil()
	}

	if v1.Kind() != reflect.Ptr || v2.Kind() != reflect.Ptr {
		// Return false if either is not a pointer
		return false
	}

	// Dereference the pointers and compare their values using reflect.DeepEqual
	return reflect.DeepEqual(v1.Elem().Interface(), v2.Elem().Interface())
}

// DoesFuzzyMatch determines whether two strings are within a specified Damerau-Levenshtein distance.
// https://en.wikipedia.org/wiki/Damerau%E2%80%93Levenshtein_distance
// Parameters:
// - source: The original string.
// - target: The string to compare against.
// - maxOperations: The maximum number of allowed operations.
func DoesFuzzyMatch(source, target string, maxOperations int) bool {
	sl := strings.ToLower(source)
	tl := strings.ToLower(target)

	// remove whitespaces if there are any
	sl = strings.ReplaceAll(sl, " ", "")
	tl = strings.ReplaceAll(tl, " ", "")

	// if either of string is empty, donot proceed
	if sl == "" || tl == "" {
		return false
	}

	// if both matches, donot proceed
	if sl == tl {
		return true
	}

	lenS := len(sl)
	lenT := len(tl)

	// if strings length difference is greater than max distance, donot proceed
	if int(math.Abs(float64(lenS-lenT))) > maxOperations {
		return false
	}

	dpm := make([][]int, lenS+1)
	for i := range dpm {
		dpm[i] = make([]int, lenT+1)
	}

	for i := 0; i <= lenS; i++ {
		dpm[i][0] = i
	}
	for j := 0; j <= lenT; j++ {
		dpm[0][j] = j
	}

	/*
		* for source phil and target pihl, dpm would look something like this after initialization
		* dpm =
			''	p	h	i	l
		''	0	1	2	3	4
		p	1	0	0	0	0
		i	2	0	0	0	0
		h	3	0	0	0	0
		l	4	0	0	0	0
	*/

	for i := 1; i <= lenS; i++ {
		for j := 1; j <= lenT; j++ {
			cost := 0
			if sl[i-1] != tl[j-1] {
				cost = 1
			}

			ins := dpm[i][j-1] + 1      // insertion
			del := dpm[i-1][j] + 1      // deletion
			sub := dpm[i-1][j-1] + cost // substitution

			dpm[i][j] = min(del, ins, sub)

			// transposition: check if matches by swapping characters in source & target
			if i > 1 && j > 1 &&
				sl[i-1] == tl[j-2] &&
				sl[i-2] == tl[j-1] {
				dpm[i][j] = min(dpm[i][j], dpm[i-2][j-2]+1)
			}
		}
	}

	/*
		* for source phil and target pihl, dpm would look something like this after Damerau–Levenshtein distance algorithm
		* dpm =
			''	p	h	i	l
		''	0	1	2	3	4
		p	1	0	1	2	3
		i	2	1	1	1	2
		h	3	2	1	1	2
		l	4	3	2	2	1
	*/

	// compare  with final total operations required
	return dpm[lenS][lenT] <= maxOperations
}
