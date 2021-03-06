package utils_test

import (
	"testing"
	"time"

	"github.com/nyaruka/goflow/utils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var laTZ, _ = time.LoadLocation("America/Los_Angeles")

var timeTests = []struct {
	DateFormat utils.DateFormat
	TimeFormat utils.TimeFormat
	Timezone   string
	FillTime   bool
	Value      string
	Expected   string
	Error      bool
}{
	// valid cases, varying formats
	{utils.DateFormatDayMonthYear, utils.TimeFormatHourMinute, "UTC", false, "01-02-2001", "01-02-2001 00:00:00 +0000 UTC", false},
	{utils.DateFormatDayMonthYear, utils.TimeFormatHourMinute, "UTC", false, "date is 01.02.2001 yes", "01-02-2001 00:00:00 +0000 UTC", false},
	{utils.DateFormatDayMonthYear, utils.TimeFormatHourMinute, "UTC", false, "date is 1-2-99 yes", "01-02-1999 00:00:00 +0000 UTC", false},
	{utils.DateFormatDayMonthYear, utils.TimeFormatHourMinute, "UTC", false, "01/02/2001", "01-02-2001 00:00:00 +0000 UTC", false},

	// must be strict iso to match despite format
	{utils.DateFormatDayMonthYear, utils.TimeFormatHourMinute, "UTC", false, "2001-01-02T10:34:56Z", "02-01-2001 10:34:56 +0000 UTC", false},
	{utils.DateFormatDayMonthYear, utils.TimeFormatHourMinute, "UTC", false, "2001-01-02T10:34:56+02:00", "02-01-2001 10:34:56 +0200 MST", false},
	{utils.DateFormatDayMonthYear, utils.TimeFormatHourMinute, "UTC", false, "2001-01-02", "02-01-2001 00:00:00 +0000 UTC", false},
	{utils.DateFormatDayMonthYear, utils.TimeFormatHourMinute, "UTC", true, "2001-01-02", "02-01-2001 13:36:30.123456789 +0000 UTC", false},
	{utils.DateFormatDayMonthYear, utils.TimeFormatHourMinute, "America/Los_Angeles", true, "2001-01-02", "02-01-2001 06:36:30.123456789 -0800 PST", false},
	{utils.DateFormatDayMonthYear, utils.TimeFormatHourMinute, "UTC", false, " 2001-01-02 ", "02-01-2001 00:00:00 +0000 UTC", false},
	{utils.DateFormatDayMonthYear, utils.TimeFormatHourMinute, "UTC", false, "on 2001-01-02 ", "", true},
	{utils.DateFormatDayMonthYear, utils.TimeFormatHourMinute, "UTC", false, "2001_01_02", "", true},
	{utils.DateFormatDayMonthYear, utils.TimeFormatHourMinute, "UTC", false, "2001-1-2", "", true},

	// month first
	{utils.DateFormatMonthDayYear, utils.TimeFormatHourMinute, "UTC", false, "01-02-2001", "02-01-2001 00:00:00 +0000 UTC", false},
	{utils.DateFormatMonthDayYear, utils.TimeFormatHourMinute, "UTC", false, "2001-01-02", "02-01-2001 00:00:00 +0000 UTC", false},
	{utils.DateFormatMonthDayYear, utils.TimeFormatHourMinute, "UTC", false, "2001-1-2", "", true},

	// year first
	{utils.DateFormatYearMonthDay, utils.TimeFormatHourMinute, "UTC", false, "2001-02-01", "01-02-2001 00:00:00 +0000 UTC", false},
	{utils.DateFormatYearMonthDay, utils.TimeFormatHourMinute, "UTC", false, "99-02-01", "01-02-1999 00:00:00 +0000 UTC", false},

	// specific timezone
	{utils.DateFormatDayMonthYear, utils.TimeFormatHourMinute, "America/Los_Angeles", false, "01\\02\\2001", "01-02-2001 00:00:00 -0800 PST", false},

	// with time filling
	{utils.DateFormatDayMonthYear, utils.TimeFormatHourMinute, "UTC", true, "01-02-2001", "01-02-2001 13:36:30.123456789 +0000 UTC", false},
	{utils.DateFormatDayMonthYear, utils.TimeFormatHourMinute, "UTC", true, "01-02-2001 04:23", "01-02-2001 04:23:00 +0000 UTC", false},
	{utils.DateFormatDayMonthYear, utils.TimeFormatHourMinute, "America/Los_Angeles", true, "01-02-2001", "01-02-2001 06:36:30.123456789 -0800 PST", false},

	// illegal day
	{utils.DateFormatDayMonthYear, utils.TimeFormatHourMinute, "UTC", false, "33-01-2001", "01-01-0001 00:00:00 +0000 UTC", true},

	// illegal month
	{utils.DateFormatDayMonthYear, utils.TimeFormatHourMinute, "UTC", false, "01-13-2001", "01-01-0001 00:00:00 +0000 UTC", true},

	// valid two digit cases
	{utils.DateFormatDayMonthYear, utils.TimeFormatHourMinute, "UTC", false, "01-01-99", "01-01-1999 00:00:00 +0000 UTC", false},
	{utils.DateFormatDayMonthYear, utils.TimeFormatHourMinute, "UTC", false, "01-01-16", "01-01-2016 00:00:00 +0000 UTC", false},
	{utils.DateFormatDayMonthYear, utils.TimeFormatHourMinute, "UTC", false, "01-01-16a", "", true},

	// iso dates
	{utils.DateFormatDayMonthYear, utils.TimeFormatHourMinute, "UTC", false, "2016-05-01T18:30:15-08:00", "01-05-2016 18:30:15 -0800 PST", false},
	{utils.DateFormatDayMonthYear, utils.TimeFormatHourMinute, "UTC", false, "2016-05-01T18:30:15Z", "01-05-2016 18:30:15 +0000 UTC", false},
	{utils.DateFormatDayMonthYear, utils.TimeFormatHourMinute, "UTC", false, "2016-05-01T18:30:15.250Z", "01-05-2016 18:30:15.250 +0000 UTC", false},
	{utils.DateFormatDayMonthYear, utils.TimeFormatHourMinute, "UTC", false, "1977-06-23T08:34:00.000-07:00", "23-06-1977 15:34:00.000 +0000 UTC", false},
	{utils.DateFormatDayMonthYear, utils.TimeFormatHourMinute, "UTC", false, "1977-06-23T08:34:00.000250-07:00", "23-06-1977 15:34:00.000250 +0000 UTC", false},
	{utils.DateFormatDayMonthYear, utils.TimeFormatHourMinute, "UTC", false, "1977-06-23T08:34:00.000250500-07:00", "23-06-1977 15:34:00.000250500 +0000 UTC", false},
	{utils.DateFormatDayMonthYear, utils.TimeFormatHourMinute, "UTC", false, "2017-06-10T17:34-06:00", "10-06-2017 23:34:00.000000 +0000 UTC", false},

	// with time
	{utils.DateFormatYearMonthDay, utils.TimeFormatHourMinute, "UTC", false, "2001-02-01 03:15", "01-02-2001 03:15:00 +0000 UTC", false},
	{utils.DateFormatYearMonthDay, utils.TimeFormatHourMinute, "UTC", false, "2001-02-01 03:15pm", "01-02-2001 15:15:00 +0000 UTC", false},
	{utils.DateFormatYearMonthDay, utils.TimeFormatHourMinute, "UTC", false, "2001-02-01 03:15 AM", "01-02-2001 03:15:00 +0000 UTC", false},
	{utils.DateFormatYearMonthDay, utils.TimeFormatHourMinute, "UTC", false, "2001-02-01 03:15:34", "01-02-2001 03:15:34 +0000 UTC", false},
	{utils.DateFormatYearMonthDay, utils.TimeFormatHourMinute, "UTC", false, "2001-02-01 03:15:34.123", "01-02-2001 03:15:34.123 +0000 UTC", false},
	{utils.DateFormatYearMonthDay, utils.TimeFormatHourMinute, "UTC", false, "2001-02-01 03:15:34.123456", "01-02-2001 03:15:34.123456 +0000 UTC", false},
}

func TestDateFromString(t *testing.T) {
	utils.SetTimeSource(utils.NewFixedTimeSource(time.Date(2018, 9, 13, 13, 36, 30, 123456789, time.UTC)))
	defer utils.SetTimeSource(utils.DefaultTimeSource)

	for _, test := range timeTests {
		timezone, err := time.LoadLocation(test.Timezone)
		require.NoError(t, err)

		env := utils.NewEnvironment(test.DateFormat, test.TimeFormat, timezone, utils.NilLanguage, nil, utils.NilCountry, utils.DefaultNumberFormat, utils.RedactionPolicyNone)

		value, err := utils.DateFromString(env, test.Value, test.FillTime)

		if test.Error {
			assert.Error(t, err)
		} else {
			require.NoError(t, err, "error parsing date %s", test.Value)

			expected, err := time.Parse("02-01-2006 15:04:05.999999999 -0700 MST", test.Expected)
			require.NoError(t, err, "error parsing expected date %s", test.Expected)

			if !expected.Equal(value) {
				assert.Fail(t, "", "mismatch for date input %s, expected %s, got %s", test.Value, expected, value)
			}
		}
	}
}

func TestDaysBetween(t *testing.T) {
	daysBetweenTests := []struct {
		d1       time.Time
		d2       time.Time
		expected int
	}{
		{time.Date(2018, 1, 1, 12, 30, 0, 0, time.UTC), time.Date(2018, 1, 3, 0, 30, 0, 0, time.UTC), -2},
		{time.Date(2018, 1, 1, 12, 30, 0, 0, time.UTC), time.Date(2018, 1, 3, 0, 30, 0, 0, laTZ), -2},
		{time.Date(2018, 1, 1, 12, 30, 0, 0, time.UTC), time.Date(2017, 12, 25, 0, 30, 0, 0, time.UTC), 7},
		{time.Date(2018, 1, 1, 12, 30, 0, 0, time.UTC), time.Date(2018, 1, 1, 12, 30, 0, 0, time.UTC), 0},
	}

	for _, test := range daysBetweenTests {
		actual := utils.DaysBetween(test.d1, test.d2)
		assert.Equal(t, test.expected, actual, "mismatch for inputs %s - %s", test.d1, test.d2)
	}
}

func TestMonthsBetween(t *testing.T) {
	monthsBetweenTests := []struct {
		d1       time.Time
		d2       time.Time
		expected int
	}{
		{time.Date(2018, 1, 1, 12, 30, 0, 0, time.UTC), time.Date(2018, 1, 3, 0, 30, 0, 0, time.UTC), 0},
		{time.Date(2018, 1, 1, 12, 30, 0, 0, time.UTC), time.Date(2018, 3, 3, 0, 30, 0, 0, laTZ), -2},
		{time.Date(2018, 1, 1, 12, 30, 0, 0, time.UTC), time.Date(2017, 12, 25, 0, 30, 0, 0, time.UTC), 1},
		{time.Date(2018, 1, 1, 12, 30, 0, 0, time.UTC), time.Date(2018, 1, 1, 12, 30, 0, 0, time.UTC), 0},
	}

	for _, test := range monthsBetweenTests {
		actual := utils.MonthsBetween(test.d1, test.d2)
		assert.Equal(t, test.expected, actual, "mismatch for inputs %s - %s", test.d1, test.d2)
	}
}

func TestDateFormat(t *testing.T) {
	formatTests := []struct {
		input    string
		expected string
		hasErr   bool
	}{
		{"MM-DD-YYYY", "01-02-2006", false},
		{"M-D-YY", "1-2-06", false},
		{"h:m", "3:4", false},
		{"h:m:s aa", "3:4:5 pm", false},
		{"h:m:s AA", "3:4:5 PM", false},
		{"YYYY-MM-DDTtt:mm:ssZZZ", "2006-01-02T15:04:05-07:00", false},
		{"YYYY-MM-DDTtt:mm:ssZZZ", "2006-01-02T15:04:05-07:00", false},
		{"YYYY-MM-DDThh:mm:ss.fffZZZ", "2006-01-02T03:04:05.000-07:00", false},
		{"YYYY-MM-DDThh:mm:ss.fffZ", "2006-01-02T03:04:05.000Z07:00", false},
		{"YYYY-MM-DD", "2006-01-02", false},

		{"tt:mm:ss.ffffff", "15:04:05.000000", false},
		{"tt:mm:ss.fffffffff", "15:04:05.000000000", false},

		{"tt:mm:ss.ffff", "", true},
		{"t:mm:ss.ffff", "", true},
		{"tt:mmm:ss.ffff", "", true},
		{"YYYY-MMM-DD", "", true},
		{"YYY-MM-DD", "", true},
		{"tt:mm:sss", "", true},
		{"tt:mm:ss a", "", true},
		{"tt:mm:ss A", "", true},
		{"tt:mm:ssZZZZ", "", true},

		{"2006-01-02", "", true},
	}

	for _, test := range formatTests {
		actual, err := utils.ToGoDateFormat(test.input, utils.DateTimeFormatting)
		if actual != test.expected {
			t.Errorf("Date format invalid for '%s'  Expected: '%s' Got: '%s'", test.input, test.expected, actual)
		}

		if err != nil && !test.hasErr {
			t.Errorf("Date format received error for '%s': %s", test.input, err)
		}

		if err == nil && test.hasErr {
			t.Errorf("Date format expected error for '%s'", test.input)
		}
	}
}
