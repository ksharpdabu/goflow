package waits

import (
	"github.com/nyaruka/goflow/flows"
	"github.com/nyaruka/goflow/flows/events"
)

func init() {
	RegisterType(TypeNothing, func() flows.Wait { return &NothingWait{} })
}

const TypeNothing string = "nothing"

// NothingWait is a wait which waits for nothing.. i.e. a chance for the caller to do
// something and resume immediately
type NothingWait struct {
	baseWait
}

// Type returns the type of this wait
func (w *NothingWait) Type() string { return TypeNothing }

// Begin beings waiting at this wait
func (w *NothingWait) Begin(run flows.FlowRun, step flows.Step) {
	w.baseWait.Begin(run)

	run.AddEvent(step, nil, events.NewNothingWait())
}

// CanResume always returns true for a nothing wait because it's not waiting for anything
func (w *NothingWait) CanResume(callerEvents []flows.CallerEvent) bool {
	return true
}

var _ flows.Wait = (*NothingWait)(nil)
