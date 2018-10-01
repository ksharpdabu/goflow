package resumes

import (
	"encoding/json"

	"github.com/nyaruka/goflow/flows"
	"github.com/nyaruka/goflow/utils"
)

func init() {
	RegisterType(TypeMsg, ReadMsgResume)
}

// TypeMsg is the type for resuming a session with a message
const TypeMsg string = "msg"

// MsgResume is used when a session is resumed with a new message from the contact
//
//   {
//     "type": "msg",
//     "contact": {
//       "uuid": "9f7ede93-4b16-4692-80ad-b7dc54a1cd81",
//       "name": "Bob",
//       "language": "fra",
//       "fields": {"gender": {"text": "Male"}},
//       "groups": []
//     },
//     "msg": {
//       "uuid": "2d611e17-fb22-457f-b802-b8f7ec5cda5b",
//       "channel": {"uuid": "61602f3e-f603-4c70-8a8f-c477505bf4bf", "name": "Twilio"},
//       "urn": "tel:+12065551212",
//       "text": "hi there",
//       "attachments": ["https://s3.amazon.com/mybucket/attachment.jpg"]
//     }
//   }
//
// @resume msg
type MsgResume struct {
	baseResume
	Msg *flows.MsgIn
}

// NewMsgResume creates a new message resume with the passed in values
func NewMsgResume(env utils.Environment, contact *flows.Contact, msg *flows.MsgIn) *MsgResume {
	return &MsgResume{
		baseResume: baseResume{
			environment: env,
			contact:     contact,
		},
		Msg: msg,
	}
}

// Type returns the type of this resume
func (t *MsgResume) Type() string { return TypeMsg }

var _ flows.Resume = (*MsgResume)(nil)

//------------------------------------------------------------------------------------------
// JSON Encoding / Decoding
//------------------------------------------------------------------------------------------

type msgResumeEnvelope struct {
	baseResumeEnvelope
	Msg *flows.MsgIn `json:"msg" validate:"required,dive"`
}

// ReadMsgResume reads a message resume
func ReadMsgResume(session flows.Session, data json.RawMessage) (flows.Resume, error) {
	resume := &MsgResume{}
	e := msgResumeEnvelope{}
	if err := utils.UnmarshalAndValidate(data, &e); err != nil {
		return nil, err
	}

	if err := unmarshalBaseResume(session, &resume.baseResume, &e.baseResumeEnvelope); err != nil {
		return nil, err
	}

	resume.Msg = e.Msg

	return resume, nil
}
