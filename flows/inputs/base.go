package inputs

import (
	"fmt"
	"time"

	"github.com/nyaruka/goflow/flows"
)

type baseInput struct {
	uuid      flows.InputUUID
	channel   flows.Channel
	createdOn time.Time
}

func (i *baseInput) UUID() flows.InputUUID  { return i.uuid }
func (i *baseInput) Channel() flows.Channel { return i.channel }
func (i *baseInput) CreatedOn() time.Time   { return i.createdOn }

// Resolve resolves the passed in key to a value, returning an error if the key is unknown
func (i *baseInput) Resolve(key string) interface{} {
	switch key {
	case "uuid":
		return i.uuid
	case "created_on":
		return i.createdOn
	case "channel":
		return i.channel
	}

	return fmt.Errorf("No such field '%s' on input", key)
}
