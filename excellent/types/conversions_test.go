package types_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/nyaruka/goflow/excellent/types"
	"github.com/nyaruka/goflow/utils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testXObject struct {
	foo string
	bar int
}

func NewTestXObject(foo string, bar int) *testXObject {
	return &testXObject{foo: foo, bar: bar}
}

// ToXJSON is called when this type is passed to @(json(...))
func (v *testXObject) ToXJSON() types.XString {
	return types.NewXMap(map[string]types.XValue{
		"foo": types.NewXString(v.foo),
		"bar": types.NewXNumberFromInt(v.bar),
	}).ToXJSON()
}

// MarshalJSON converts this type to its internal JSON representation which can differ from ToJSON
func (v *testXObject) MarshalJSON() ([]byte, error) {
	e := struct {
		Foo string `json:"foo"`
	}{
		Foo: v.foo,
	}
	return json.Marshal(e)
}

func (v *testXObject) Reduce() types.XPrimitive { return types.NewXString(v.foo) }

var _ types.XValue = &testXObject{}

func TestXValueRequiredConversions(t *testing.T) {
	chi, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)

	date1 := time.Date(2017, 6, 23, 15, 30, 0, 0, time.UTC)
	date2 := time.Date(2017, 7, 18, 15, 30, 0, 0, chi)

	tests := []struct {
		value          types.XValue
		asJSON         string
		asInternalJSON string
		asString       string
		asBool         bool
	}{
		{
			value:          nil,
			asInternalJSON: `null`,
			asJSON:         `null`,
			asString:       "",
			asBool:         false,
		}, {
			value:          types.NewXString(""),
			asInternalJSON: `""`,
			asJSON:         `""`,
			asString:       "",
			asBool:         false, // empty strings are false
		}, {
			value:          types.NewXString("FALSE"),
			asInternalJSON: `"FALSE"`,
			asJSON:         `"FALSE"`,
			asString:       "FALSE",
			asBool:         false, // because it's string value is "false"
		}, {
			value:          types.NewXString("hello \"bob\""),
			asInternalJSON: `"hello \"bob\""`,
			asJSON:         `"hello \"bob\""`,
			asString:       "hello \"bob\"",
			asBool:         true,
		}, {
			value:          types.NewXNumberFromInt(0),
			asInternalJSON: `0`,
			asJSON:         `0`,
			asString:       "0",
			asBool:         false, // because any decimal != 0 is true
		}, {
			value:          types.NewXNumberFromInt(123),
			asInternalJSON: `123`,
			asJSON:         `123`,
			asString:       "123",
			asBool:         true, // because any decimal != 0 is true
		}, {
			value:          types.RequireXNumberFromString("123.00"),
			asInternalJSON: `123`,
			asJSON:         `123`,
			asString:       "123",
			asBool:         true,
		}, {
			value:          types.RequireXNumberFromString("123.45"),
			asInternalJSON: `123.45`,
			asJSON:         `123.45`,
			asString:       "123.45",
			asBool:         true,
		}, {
			value:          types.NewXBool(false),
			asInternalJSON: `false`,
			asJSON:         `false`,
			asString:       "false",
			asBool:         false,
		}, {
			value:          types.NewXBool(true),
			asInternalJSON: `true`,
			asJSON:         `true`,
			asString:       "true",
			asBool:         true,
		}, {
			value:          types.NewXDate(date1),
			asInternalJSON: `"2017-06-23T15:30:00Z"`,
			asJSON:         `"2017-06-23T15:30:00.000000Z"`,
			asString:       "2017-06-23T15:30:00.000000Z",
			asBool:         true,
		}, {
			value:          types.NewXDate(date2),
			asInternalJSON: `"2017-07-18T15:30:00-05:00"`,
			asJSON:         `"2017-07-18T15:30:00.000000-05:00"`,
			asString:       "2017-07-18T15:30:00.000000-05:00",
			asBool:         true,
		}, {
			value:          types.NewXArray(),
			asInternalJSON: `[]`,
			asJSON:         `[]`,
			asString:       `[]`,
			asBool:         false,
		}, {
			value:          types.NewXArray(types.NewXDate(date1), types.NewXDate(date2)),
			asInternalJSON: `["2017-06-23T15:30:00Z","2017-07-18T15:30:00-05:00"]`,
			asJSON:         `["2017-06-23T15:30:00.000000Z","2017-07-18T15:30:00.000000-05:00"]`,
			asString:       `["2017-06-23T15:30:00.000000Z","2017-07-18T15:30:00.000000-05:00"]`,
			asBool:         true,
		}, {
			value:          NewTestXObject("Hello", 123),
			asInternalJSON: `{"foo":"Hello"}`,
			asJSON:         `{"bar":123,"foo":"Hello"}`,
			asString:       "Hello",
			asBool:         true,
		}, {
			value:          NewTestXObject("", 123),
			asInternalJSON: `{"foo":""}`,
			asJSON:         `{"bar":123,"foo":""}`,
			asString:       "",
			asBool:         false, // because it reduces to a string which itself is false
		}, {
			value:          types.NewXArray(NewTestXObject("Hello", 123), NewTestXObject("World", 456)),
			asInternalJSON: `[{"foo":"Hello"},{"foo":"World"}]`,
			asJSON:         `[{"bar":123,"foo":"Hello"},{"bar":456,"foo":"World"}]`,
			asString:       `["Hello","World"]`,
			asBool:         true,
		}, {
			value: types.NewXMap(map[string]types.XValue{
				"first":  NewTestXObject("Hello", 123),
				"second": NewTestXObject("World", 456),
			}),
			asInternalJSON: `{"first":{"foo":"Hello"},"second":{"foo":"World"}}`,
			asJSON:         `{"first":{"bar":123,"foo":"Hello"},"second":{"bar":456,"foo":"World"}}`,
			asString:       `{"first":"Hello","second":"World"}`,
			asBool:         true,
		}, {
			value:          types.NewXJSONArray([]byte(`[]`)),
			asInternalJSON: `[]`,
			asJSON:         `[]`,
			asString:       `[]`,
			asBool:         false,
		}, {
			value:          types.NewXJSONArray([]byte(`[5,     "x"]`)),
			asInternalJSON: `[5,"x"]`,
			asJSON:         `[5,     "x"]`,
			asString:       `[5,     "x"]`,
			asBool:         true,
		}, {
			value:          types.NewXJSONObject([]byte(`{}`)),
			asInternalJSON: `{}`,
			asJSON:         `{}`,
			asString:       `{}`,
			asBool:         false,
		}, {
			value:          types.NewXJSONObject([]byte(`{"foo":"World","bar":456}`)),
			asInternalJSON: `{"foo":"World","bar":456}`,
			asJSON:         `{"foo":"World","bar":456}`,
			asString:       `{"foo":"World","bar":456}`,
			asBool:         true,
		}, {
			value:          types.NewXError(fmt.Errorf("it failed")), // once an error, always an error
			asInternalJSON: "",
			asJSON:         "",
			asString:       "",
			asBool:         false,
		},
	}
	for _, test := range tests {
		asInternalJSON, _ := json.Marshal(test.value)
		asJSON, _ := types.ToXJSON(test.value)
		asString, _ := types.ToXString(test.value)
		asBool, _ := types.ToXBool(test.value)

		assert.Equal(t, test.asInternalJSON, string(asInternalJSON), "json.Marshal failed for %T{%s}", test.value, test.value)
		assert.Equal(t, types.NewXString(test.asJSON), asJSON, "ToXJSON failed for %T{%s}", test.value, test.value)
		assert.Equal(t, types.NewXString(test.asString), asString, "ToXString failed for %T{%s}", test.value, test.value)
		assert.Equal(t, types.NewXBool(test.asBool), asBool, "ToXBool failed for %T{%s}", test.value, test.value)
	}
}

func TestToXNumber(t *testing.T) {
	var tests = []struct {
		value    types.XValue
		asNumber types.XNumber
		hasError bool
	}{
		{nil, types.XNumberZero, false},
		{types.NewXErrorf("Error"), types.XNumberZero, true},
		{types.NewXNumberFromInt(123), types.NewXNumberFromInt(123), false},
		{types.NewXString("15.5"), types.RequireXNumberFromString("15.5"), false},
		{types.NewXString("lO.5"), types.RequireXNumberFromString("10.5"), false},
		{NewTestXObject("Hello", 123), types.XNumberZero, true},
		{NewTestXObject("123.45000", 123), types.RequireXNumberFromString("123.45"), false},
	}

	for _, test := range tests {
		result, err := types.ToXNumber(test.value)

		if test.hasError {
			assert.Error(t, err, "expected error for input %T{%s}", test.value, test.value)
		} else {
			assert.NoError(t, err, "unexpected error for input %T{%s}", test.value, test.value)
			assert.Equal(t, test.asNumber.Native(), result.Native(), "result mismatch for input %T{%s}", test.value, test.value)
		}
	}
}

func TestToXDate(t *testing.T) {
	var tests = []struct {
		value    types.XValue
		asNumber types.XDate
		hasError bool
	}{
		{nil, types.XDateZero, false},
		{types.NewXError(fmt.Errorf("Error")), types.XDateZero, true},
		{types.NewXNumberFromInt(123), types.XDateZero, true},
		{types.NewXString("2018-06-05"), types.NewXDate(time.Date(2018, 6, 5, 0, 0, 0, 0, time.UTC)), false},
		{types.NewXString("wha?"), types.XDateZero, true},
		{NewTestXObject("Hello", 123), types.XDateZero, true},
		{NewTestXObject("2018/6/5", 123), types.NewXDate(time.Date(2018, 6, 5, 0, 0, 0, 0, time.UTC)), false},
	}

	env := utils.NewDefaultEnvironment()

	for _, test := range tests {
		result, err := types.ToXDate(env, test.value)

		if test.hasError {
			assert.Error(t, err, "expected error for input %T{%s}", test.value, test.value)
		} else {
			assert.NoError(t, err, "unexpected error for input %T{%s}", test.value, test.value)
			assert.Equal(t, test.asNumber.Native(), result.Native(), "result mismatch for input %T{%s}", test.value, test.value)
		}
	}
}