package util

import (
	"testing"
	"time"
)

// TestDateStartTime_UTCPSTDateDivergence pins the contract difference between the two
// day-start helpers, which diverge for any timestamp in the midnight-7AM UTC window
// (midnight-8AM during standard time), where the UTC calendar day is already one day
// ahead of the PST calendar day.
//
//	DateStartTime    truncates the wall-clock fields as they already read on the input,
//	                 then stamps them with loc. It never converts, so it is only correct
//	                 when the input is already in loc.
//	DateStartTimePST converts to PST first, then truncates.
//
// Handing DateStartTime a UTC-based timestamp along with LocationPST therefore yields the
// UTC calendar day wearing a Pacific offset.
func TestDateStartTime_UTCPSTDateDivergence(t *testing.T) {
	// July 7, 2026 01:12 UTC == July 6, 2026 18:12 PDT
	ts := time.Date(2026, time.July, 7, 1, 12, 0, 0, time.UTC)

	// No conversion: the input's UTC fields (July 7) are reused as Pacific wall-clock.
	wantUTCDayStart := time.Date(2026, time.July, 7, 0, 0, 0, 0, LocationPST)
	gotUTCDayStart := DateStartTime(&ts, LocationPST)
	if !gotUTCDayStart.Equal(wantUTCDayStart) {
		t.Errorf("DateStartTime(%s, LocationPST) = %s; want %s (UTC fields reused as Pacific wall-clock)",
			ts.Format(time.RFC3339), gotUTCDayStart.Format(time.RFC3339), wantUTCDayStart.Format(time.RFC3339))
	}

	// Converts first, so it anchors to the Pacific day the timestamp actually falls on.
	wantPSTDayStart := time.Date(2026, time.July, 6, 0, 0, 0, 0, LocationPST)
	gotPSTDayStart := DateStartTimePST(&ts)
	if !gotPSTDayStart.Equal(wantPSTDayStart) {
		t.Errorf("DateStartTimePST(%s) = %s; want %s (Pacific day the timestamp falls on)",
			ts.Format(time.RFC3339), gotPSTDayStart.Format(time.RFC3339), wantPSTDayStart.Format(time.RFC3339))
	}

	// State the skipped day outright. July has no DST transition, so these consecutive
	// midnights are exactly 24h apart; a DST-straddling pair would be 23h or 25h.
	if diff := gotUTCDayStart.Sub(gotPSTDayStart); diff != 24*time.Hour {
		t.Errorf("DateStartTime anchored %s ahead of DateStartTimePST; want exactly %s",
			diff, 24*time.Hour)
	}
}
