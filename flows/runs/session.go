package runs

import (
	"encoding/json"

	"github.com/nyaruka/goflow/flows"
	"github.com/nyaruka/goflow/utils"
)

type session struct {
	runs   []flows.FlowRun
	events []flows.EventLogEntry
}

func newSession() *session {
	session := session{}
	return &session
}

func (s *session) AddRun(run flows.FlowRun) {
	// check if we already have this run
	for _, r := range s.runs {
		if r.UUID() == run.UUID() {
			return
		}
	}
	s.runs = append(s.runs, run)
}
func (s *session) Runs() []flows.FlowRun { return s.runs }

func (s *session) ActiveRun() flows.FlowRun {
	var active flows.FlowRun
	mostRecent := utils.ZeroTime

	for _, run := range s.runs {
		// We are complete, therefore can't be active
		if run.IsComplete() {
			continue
		}

		// We have a child, and it isn't complete, we can't be active
		if run.Child() != nil && run.Child().Status() == flows.StatusActive {
			continue
		}

		// this is more recent than our most recent flow
		if run.ModifiedOn().After(mostRecent) {
			active = run
			mostRecent = run.ModifiedOn()
		}
	}
	return active
}

func (s *session) LogEvent(step flows.Step, action flows.Action, event flows.Event) {
	s.events = append(s.events, NewEventLogEntry(step, action, event))
}
func (s *session) EventLog() []flows.EventLogEntry { return s.events }
func (s *session) ClearEventLog()                  { s.events = nil }

//------------------------------------------------------------------------------------------
// JSON Encoding / Decoding
//------------------------------------------------------------------------------------------

// ReadSession decodes a session from the passed in JSON
func ReadSession(data json.RawMessage) (flows.Session, error) {
	session := &session{}
	err := json.Unmarshal(data, session)
	if err == nil {
		err = utils.ValidateAll(session)
	}
	return session, err
}

type sessionEnvelope struct {
	Runs []*flowRun `json:"runs"`
}

func (s *session) UnmarshalJSON(data []byte) error {
	var se sessionEnvelope
	var err error

	err = json.Unmarshal(data, &se)
	if err != nil {
		return err
	}

	s.runs = make([]flows.FlowRun, len(se.Runs))
	for i := range s.runs {
		s.runs[i] = se.Runs[i]
		s.runs[i].SetSession(s)
	}
	return nil
}

func (s *session) MarshalJSON() ([]byte, error) {
	var se sessionEnvelope
	se.Runs = make([]*flowRun, len(s.runs))
	for i := range s.runs {
		se.Runs[i] = s.runs[i].(*flowRun)
	}

	return json.Marshal(se)
}
