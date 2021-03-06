package events

import (
	"time"

	"github.com/nyaruka/goflow/flows"
)

func init() {
	RegisterType(TypeMsgWait, func() flows.Event { return &MsgWaitEvent{} })
}

// TypeMsgWait is the type of our msg wait event
const TypeMsgWait string = "msg_wait"

// MsgWaitEvent events are created when a flow pauses waiting for a response from
// a contact. If a timeout is set, then the caller should resume the flow after
// the number of seconds in the timeout to resume it.
//
//   {
//     "type": "msg_wait",
//     "created_on": "2006-01-02T15:04:05Z",
//     "timeout": 300
//   }
//
// @event msg_wait
type MsgWaitEvent struct {
	BaseEvent

	TimeoutOn *time.Time `json:"timeout_on,omitempty"`
}

// NewMsgWait returns a new msg wait with the passed in timeout
func NewMsgWait(timeoutOn *time.Time) *MsgWaitEvent {
	return &MsgWaitEvent{
		BaseEvent: NewBaseEvent(TypeMsgWait),
		TimeoutOn: timeoutOn,
	}
}
