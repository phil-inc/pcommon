package util

import "time"

var timeZoneToLocationMap = make(map[string]*time.Location)

// LocationPST location America/Los_Angeles
var LocationPST *time.Location

// LocationEST location America/New_York
var LocationEST *time.Location

const localDateFormat = "2006-01-02"    // yyyy-mm-dd
const MMDDYYYYDateFormat = "01/02/2006" // mm/dd/yyyy
const localDateFormatWithTime = "2006-01-02 15:04 MST"
const PstTimeZone = "America/Los_Angeles"
const EstTimeZone = "America/New_York"
const CstTimeZone = "America/Chicago"
const MstTimeZone = "America/Phoenix"
const YYYYMMDDFormater = "20060102"
const LongFormat = "2006-01-02 15:04:05.999999999 -0700 MST"

const (
	UtmSrc            = "utm_src"
	UtmSource         = "utm_source"
	UtmSourceSMS      = "sms"
	UtmSourceMandrill = "mandrill"
)

var alphanumericRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

var alphabeticRunes = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ")

// Empty empty string
var Empty = ""

var nameSuffixes = map[string]bool{
	"jr":  true,
	"sr":  true,
	"i":   true,
	"ii":  true,
	"iii": true,
	"iv":  true,
	"v":   true,
	"md":  true,
	"dds": true,
	"phd": true,
	"dvm": true,
}

var stateToTimezoneMap = map[string]string{
	"AK": "America/Anchorage",
	"AL": CstTimeZone,
	"AR": CstTimeZone,
	"AS": "Etc/GMT+11",
	"AZ": "America/Phoenix",
	"CA": PstTimeZone,
	"CO": "America/Denver",
	"CT": EstTimeZone,
	"DC": EstTimeZone,
	"DE": EstTimeZone,
	"FL": EstTimeZone,
	"GA": EstTimeZone,
	"HI": "Pacific/Honolulu",
	"IA": CstTimeZone,
	"ID": "America/Boise",
	"IL": CstTimeZone,
	"IN": "America/Indiana/Indianapolis",
	"KS": CstTimeZone,
	"KY": "America/Kentucky/Louisville",
	"LA": CstTimeZone,
	"MA": EstTimeZone,
	"MD": EstTimeZone,
	"ME": EstTimeZone,
	"MI": "America/Detroit",
	"MN": CstTimeZone,
	"MO": CstTimeZone,
	"MS": CstTimeZone,
	"MT": "America/Denver",
	"NC": EstTimeZone,
	"ND": CstTimeZone,
	"NE": CstTimeZone,
	"NH": EstTimeZone,
	"NJ": EstTimeZone,
	"NM": "America/Denver",
	"NV": PstTimeZone,
	"NY": EstTimeZone,
	"OH": EstTimeZone,
	"OK": CstTimeZone,
	"OR": PstTimeZone,
	"PA": EstTimeZone,
	"PR": "America/Halifax",
	"RI": EstTimeZone,
	"SC": EstTimeZone,
	"SD": CstTimeZone,
	"TN": CstTimeZone,
	"TX": CstTimeZone,
	"UT": "America/Denver",
	"VA": EstTimeZone,
	"VI": EstTimeZone,
	"VT": EstTimeZone,
	"WA": PstTimeZone,
	"WI": CstTimeZone,
	"WV": EstTimeZone,
	"WY": "America/Denver",
}

// This is a map of user state to relevant one of the 4 timezone we use for running batch jobs
// CstTimeZone, PstTimeZone, EstTimeZone, "America/Phoenix"
var stateToStandardTimeZoneMap = map[string]string{
	"AK": CstTimeZone,
	"AL": CstTimeZone,
	"AR": CstTimeZone,
	"AS": CstTimeZone,
	"AZ": MstTimeZone,
	"CA": PstTimeZone,
	"CO": CstTimeZone,
	"CT": EstTimeZone,
	"DC": EstTimeZone,
	"DE": EstTimeZone,
	"FL": EstTimeZone,
	"GA": EstTimeZone,
	"HI": PstTimeZone,
	"IA": CstTimeZone,
	"ID": CstTimeZone,
	"IL": CstTimeZone,
	"IN": CstTimeZone,
	"KS": CstTimeZone,
	"KY": CstTimeZone,
	"LA": CstTimeZone,
	"MA": EstTimeZone,
	"MD": EstTimeZone,
	"ME": EstTimeZone,
	"MI": CstTimeZone,
	"MN": CstTimeZone,
	"MO": CstTimeZone,
	"MS": CstTimeZone,
	"MT": CstTimeZone,
	"NC": EstTimeZone,
	"ND": CstTimeZone,
	"NE": CstTimeZone,
	"NH": EstTimeZone,
	"NJ": EstTimeZone,
	"NM": CstTimeZone,
	"NV": PstTimeZone,
	"NY": EstTimeZone,
	"OH": EstTimeZone,
	"OK": CstTimeZone,
	"OR": PstTimeZone,
	"PA": EstTimeZone,
	"PR": EstTimeZone,
	"RI": EstTimeZone,
	"SC": EstTimeZone,
	"SD": CstTimeZone,
	"TN": CstTimeZone,
	"TX": CstTimeZone,
	"UT": CstTimeZone,
	"VA": EstTimeZone,
	"VI": EstTimeZone,
	"VT": EstTimeZone,
	"WA": PstTimeZone,
	"WI": CstTimeZone,
	"WV": EstTimeZone,
	"WY": CstTimeZone,
}

var StateAbbreviation = map[string]string{
	"AL": "Alabama",
	"AK": "Alaska",
	"AZ": "Arizona",
	"AR": "Arkansas",
	"CA": "California",
	"CO": "Colorado",
	"CT": "Connecticut",
	"DE": "Delaware",
	"FL": "Florida",
	"GA": "Georgia",
	"HI": "Hawaii",
	"ID": "Idaho",
	"IL": "Illinois",
	"IN": "Indiana",
	"IA": "Iowa",
	"KS": "Kansas",
	"KY": "Kentucky",
	"LA": "Louisiana",
	"ME": "Maine",
	"MD": "Maryland",
	"MA": "Massachusetts",
	"MI": "Michigan",
	"MN": "Minnesota",
	"MS": "Mississippi",
	"MO": "Missouri",
	"MT": "Montana",
	"NE": "Nebraska",
	"NV": "Nevada",
	"NH": "New Hampshire",
	"NJ": "New Jersey",
	"NM": "New Mexico",
	"NY": "New York",
	"NC": "North Carolina",
	"ND": "North Dakota",
	"OH": "Ohio",
	"OK": "Oklahoma",
	"OR": "Oregon",
	"PA": "Pennsylvania",
	"RI": "Rhode Island",
	"SC": "South Carolina",
	"SD": "South Dakota",
	"TN": "Tennessee",
	"TX": "Texas",
	"UT": "Utah",
	"VT": "Vermont",
	"VA": "Virginia",
	"WA": "Washington",
	"WV": "West Virginia",
	"WI": "Wisconsin",
	"WY": "Wyoming",
	// Territories
	"AS": "American Samoa",
	"DC": "District of Columbia",
	"FM": "Federated States of Micronesia",
	"GU": "Guam",
	"MH": "Marshall Islands",
	"MP": "Northern Mariana Islands",
	"PW": "Palau",
	"PR": "Puerto Rico",
	"VI": "Virgin Islands",
}

// StartOfTheWeekMap day map for the week
var StartOfTheWeekMap = map[time.Weekday]int{time.Monday: 0, time.Tuesday: 1, time.Wednesday: 2, time.Thursday: 3, time.Friday: 4, time.Saturday: 5, time.Sunday: 6}
