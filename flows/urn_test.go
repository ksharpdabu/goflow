package flows_test

import (
	"testing"
	"time"

	"github.com/nyaruka/gocommon/urns"
	"github.com/nyaruka/goflow/assets/static"
	"github.com/nyaruka/goflow/excellent"
	"github.com/nyaruka/goflow/excellent/types"
	"github.com/nyaruka/goflow/flows"
	"github.com/nyaruka/goflow/flows/engine"
	"github.com/nyaruka/goflow/utils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContactURN(t *testing.T) {
	source, err := static.NewStaticSource([]byte(`{
		"channels": [
			{
				"uuid": "57f1078f-88aa-46f4-a59a-948a5739c03d",
				"name": "Android Channel",
				"address": "+12345671111",
				"schemes": [
					"tel"
				],
				"roles": [
					"send",
					"receive"
				]
			}
	    ]
	}`))
	require.NoError(t, err)

	sessionAssets, err := engine.NewSessionAssets(source)
	require.NoError(t, err)

	channels := sessionAssets.Channels()

	channel, err := channels.Get("57f1078f-88aa-46f4-a59a-948a5739c03d")
	require.NoError(t, err)

	// check that parsing a URN properly extracts its channel affinity
	urn, err := flows.ParseRawURN(channels, urns.URN("tel:+250781234567?channel=57f1078f-88aa-46f4-a59a-948a5739c03d&id=3"))
	assert.NoError(t, err)
	assert.Equal(t, urns.URN("tel:+250781234567?channel=57f1078f-88aa-46f4-a59a-948a5739c03d&id=3"), urn.URN())
	assert.Equal(t, channel, urn.Channel())
	assert.Equal(t, "tel:+250781234567?channel=57f1078f-88aa-46f4-a59a-948a5739c03d&id=3", urn.String())

	// check equality
	urn2, _ := flows.ParseRawURN(channels, urns.URN("tel:+250781234567?channel=57f1078f-88aa-46f4-a59a-948a5739c03d&id=3"))
	urn3, _ := flows.ParseRawURN(channels, urns.URN("tel:+250781234567?id=3"))
	assert.True(t, urn.Equal(urn2))
	assert.False(t, urn.Equal(urn3))

	// check using URN in expressions
	env := utils.NewDefaultEnvironment()
	assert.Equal(t, "URN", urn.Describe())
	assert.Equal(t, types.NewXText("tel:+250781234567"), urn.Reduce(env))
	assert.Equal(t, types.NewXText("tel"), urn.Resolve(env, "scheme"))
	assert.Equal(t, types.NewXText("+250781234567"), urn.Resolve(env, "path"))
	assert.Equal(t, types.NewXText("0781 234 567"), urn.Resolve(env, "display"))
	assert.Equal(t, channel, urn.Resolve(env, "channel"))
	assert.Equal(t, types.NewXResolveError(urn, "xxx"), urn.Resolve(env, "xxx"))
	assert.Equal(t, types.NewXText(`{"display":"0781 234 567","path":"+250781234567","scheme":"tel"}`), urn.ToXJSON(env))

	// check when URNs have to be redacted
	env = utils.NewEnvironment(utils.DateFormatDayMonthYear, utils.TimeFormatHourMinute, time.UTC, utils.Language("eng"), nil, utils.NilCountry, utils.DefaultNumberFormat, utils.RedactionPolicyURNs)
	assert.Equal(t, types.NewXText("********"), urn.Reduce(env))
	assert.Equal(t, types.NewXText("tel"), urn.Resolve(env, "scheme"))
	assert.Equal(t, types.NewXText("********"), urn.Resolve(env, "path"))
	assert.Equal(t, types.NewXText("********"), urn.Resolve(env, "display"))
	assert.Equal(t, channel, urn.Resolve(env, "channel"))
	assert.Equal(t, types.NewXText(`{"display":"********","path":"********","scheme":"tel"}`), urn.ToXJSON(env))

	// we can clear the channel affinity
	urn.SetChannel(nil)
	assert.Equal(t, urns.URN("tel:+250781234567?id=3"), urn.URN())
	assert.Nil(t, urn.Channel())

	// and change it
	urn.SetChannel(channel)
	assert.Equal(t, urns.URN("tel:+250781234567?channel=57f1078f-88aa-46f4-a59a-948a5739c03d&id=3"), urn.URN())
	assert.Equal(t, channel, urn.Channel())
}

func TestURNList(t *testing.T) {
	urn1 := flows.NewContactURN("tel:+250781234567", nil)
	urn2 := flows.NewContactURN("twitter:134252511151#billy_bob", nil)
	urn3 := flows.NewContactURN("tel:+250781111222", nil)
	urnList := flows.URNList{urn1, urn2, urn3}

	env := utils.NewDefaultEnvironment()

	// check equality
	assert.True(t, urnList.Equal(flows.URNList{urn1, urn2, urn3}))
	assert.False(t, urnList.Equal(flows.URNList{urn3, urn2, urn1}))
	assert.False(t, urnList.Equal(flows.URNList{urn1, urn2}))

	// check use in expressions
	assert.Equal(t, "URNs", urnList.Describe())
	assert.Equal(t, types.NewXArray(urn1, urn2, urn3), urnList.Reduce(env))
	assert.Equal(t, 3, urnList.Length())
	assert.Equal(t, urn3, urnList.Index(2))
	assert.Equal(t, types.NewXText(`[{"display":"0781 234 567","path":"+250781234567","scheme":"tel"},{"display":"billy_bob","path":"134252511151","scheme":"twitter"},{"display":"0781 111 222","path":"+250781111222","scheme":"tel"}]`), urnList.ToXJSON(env))

	testCases := []struct {
		key      string
		hasValue bool
		value    interface{}
	}{
		{"0", true, flows.NewContactURN("tel:+250781234567", nil)},
		{"1", true, flows.NewContactURN("twitter:134252511151#billy_bob", nil)},
		{"2", true, flows.NewContactURN("tel:+250781111222", nil)},
		{"-1", true, flows.NewContactURN("tel:+250781111222", nil)},
		{"3", false, nil}, // index out of range
		{"tel", true, flows.URNList{flows.NewContactURN("tel:+250781234567", nil), flows.NewContactURN("tel:+250781111222", nil)}},
		{"twitter", true, flows.URNList{flows.NewContactURN("twitter:134252511151#billy_bob", nil)}},
		{"xxxxxx", false, ""}, // not a valid scheme
	}
	for _, tc := range testCases {
		val := excellent.ResolveValue(env, urnList, tc.key)

		err, isErr := val.(error)

		if tc.hasValue && isErr {
			t.Errorf("Got unexpected error resolving %s: %s", tc.key, err)
		}

		if !tc.hasValue && !isErr {
			t.Errorf("Did not get expected error resolving %s", tc.key)
		}

		if tc.hasValue {
			assert.Equal(t, tc.value, val)
		}
	}
}
