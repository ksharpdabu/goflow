package engine

import (
	"encoding/json"
	"fmt"

	"github.com/nyaruka/goflow/assets"
	"github.com/nyaruka/goflow/flows"
	"github.com/nyaruka/goflow/flows/actions"
	"github.com/nyaruka/goflow/flows/events"
	"github.com/nyaruka/goflow/flows/runs"
	"github.com/nyaruka/goflow/flows/triggers"
	"github.com/nyaruka/goflow/flows/waits"
	"github.com/nyaruka/goflow/utils"

	log "github.com/sirupsen/logrus"
)

// used to spawn a new run or sub-flow in the event loop
type pushedFlow struct {
	flow      flows.Flow
	parentRun flows.FlowRun
}

type session struct {
	assets flows.SessionAssets

	// state which is maintained between engine calls
	env     utils.Environment
	trigger flows.Trigger
	contact *flows.Contact
	runs    []flows.FlowRun
	status  flows.SessionStatus
	wait    flows.Wait

	// state which is temporary to each call
	runsByUUID map[flows.RunUUID]flows.FlowRun
	pushedFlow *pushedFlow
	flowStack  *flowStack
	newEvents  []flows.Event

	engineConfig flows.EngineConfig
	httpClient   *utils.HTTPClient
}

// NewSession creates a new session
func NewSession(assets flows.SessionAssets, engineConfig flows.EngineConfig, httpClient *utils.HTTPClient) flows.Session {
	return &session{
		env:          utils.NewDefaultEnvironment(),
		assets:       assets,
		status:       flows.SessionStatusActive,
		newEvents:    []flows.Event{},
		runsByUUID:   make(map[flows.RunUUID]flows.FlowRun),
		flowStack:    newFlowStack(),
		engineConfig: engineConfig,
		httpClient:   httpClient,
	}
}

func (s *session) Assets() flows.SessionAssets          { return s.assets }
func (s *session) Environment() utils.Environment       { return s.env }
func (s *session) SetEnvironment(env utils.Environment) { s.env = env }
func (s *session) Trigger() flows.Trigger               { return s.trigger }
func (s *session) Contact() *flows.Contact              { return s.contact }
func (s *session) SetContact(contact *flows.Contact)    { s.contact = contact }

func (s *session) FlowOnStack(uuid assets.FlowUUID) bool { return s.flowStack.hasFlow(uuid) }

func (s *session) PushFlow(flow flows.Flow, parentRun flows.FlowRun) {
	s.pushedFlow = &pushedFlow{flow: flow, parentRun: parentRun}
}

func (s *session) Runs() []flows.FlowRun { return s.runs }
func (s *session) GetRun(uuid flows.RunUUID) (flows.FlowRun, error) {
	run, exists := s.runsByUUID[uuid]
	if exists {
		return run, nil
	}
	return nil, fmt.Errorf("unable to find run with UUID: %s", uuid)
}

func (s *session) addRun(run flows.FlowRun) {
	s.runs = append(s.runs, run)
	s.runsByUUID[run.UUID()] = run
}

func (s *session) GetCurrentChild(run flows.FlowRun) flows.FlowRun {
	// the current child of a run, is the last added run which has that run as its parent
	for r := len(s.runs) - 1; r >= 0; r-- {
		if s.runs[r].ParentInSession() == run {
			return s.runs[r]
		}
	}
	return nil
}

// ParentRun gets the parent run of this session if it was started by a flow action
func (s *session) ParentRun() flows.RunSummary {
	if s.trigger != nil && s.trigger.Type() == triggers.TypeFlowAction {
		trigger := s.trigger.(*triggers.FlowActionTrigger)
		return trigger.Run()
	}
	return nil
}

func (s *session) Status() flows.SessionStatus { return s.status }
func (s *session) Wait() flows.Wait            { return s.wait }

// looks through this session's run for the one that is waiting
func (s *session) waitingRun() flows.FlowRun {
	for _, run := range s.runs {
		if run.Status() == flows.RunStatusWaiting {
			return run
		}
	}
	return nil
}

func (s *session) LogEvent(event flows.Event) {
	s.newEvents = append(s.newEvents, event)
}
func (s *session) Events() []flows.Event { return s.newEvents }

func (s *session) EngineConfig() flows.EngineConfig { return s.engineConfig }
func (s *session) HTTPClient() *utils.HTTPClient    { return s.httpClient }

//------------------------------------------------------------------------------------------
// Flow execution
//------------------------------------------------------------------------------------------

// Start beings processing of this session from a trigger and a set of initial caller events
func (s *session) Start(trigger flows.Trigger, callerEvents []flows.CallerEvent) error {
	// try to load the flow
	flow, err := s.Assets().Flows().Get(trigger.Flow().UUID)
	if err != nil {
		return fmt.Errorf("unable to load flow[uuid=%s]: %s", trigger.Flow().UUID, err)
	}

	// check flow is valid and has everything it needs to run
	if err := flow.Validate(s.Assets()); err != nil {
		return fmt.Errorf("validation failed for flow[uuid=%s]: %s", flow.UUID(), err)
	}

	// check caller events are valid
	if err := s.validateCallerEvents(callerEvents); err != nil {
		return err
	}

	if trigger.Environment() != nil {
		s.env = trigger.Environment()
	}
	if trigger.Contact() != nil {
		s.contact = trigger.Contact().Clone()
	}

	s.trigger = trigger
	s.PushFlow(flow, nil)

	// off to the races...
	return s.continueUntilWait(nil, noDestination, nil, callerEvents)
}

// Resume tries to resume a waiting session
func (s *session) Resume(callerEvents []flows.CallerEvent) error {
	if s.status != flows.SessionStatusWaiting {
		return fmt.Errorf("only waiting sessions can be resumed")
	}

	waitingRun := s.waitingRun()
	if waitingRun == nil {
		return fmt.Errorf("session doesn't contain any runs which are waiting")
	}

	// check flow is valid and has everything it needs to run
	if err := waitingRun.Flow().Validate(s.Assets()); err != nil {
		return fmt.Errorf("validation failed for flow[uuid=%s]: %s", waitingRun.Flow().UUID(), err)
	}

	// check caller events are valid
	if err := s.validateCallerEvents(callerEvents); err != nil {
		return err
	}

	if err := s.tryToResume(waitingRun, callerEvents); err != nil {
		// if we got an error, add it to the log and shut everything down
		for _, run := range s.runs {
			run.Exit(flows.RunStatusErrored)
		}
		s.status = flows.SessionStatusErrored
		s.LogEvent(events.NewErrorEvent(err))
	}

	return nil
}

// Resume resumes a waiting session
func (s *session) tryToResume(waitingRun flows.FlowRun, callerEvents []flows.CallerEvent) error {
	// figure out where in the flow we began waiting on
	step, _, err := waitingRun.PathLocation()
	if err != nil {
		return err
	}

	// set up our flow stack based on the current run hierarchy
	s.flowStack = flowStackFromRun(waitingRun)

	// apply and add our caller events to this step
	for _, event := range callerEvents {
		if err := event.Apply(waitingRun); err != nil {
			return err
		}
		waitingRun.AddEvent(step, nil, event)
	}

	var destination flows.NodeUUID

	// events can change run status so only proceed to the wait if we're still waiting
	if waitingRun.Status() == flows.RunStatusWaiting {
		if s.wait.CanResume(callerEvents) {
			s.wait = nil
			s.status = flows.SessionStatusActive
			waitingRun.SetStatus(flows.RunStatusActive)

			destination, err = s.findResumeDestination(waitingRun)
			if err != nil {
				return err
			}
		} else {
			// if our wait isn't satisfied, return immediately to the caller
			return nil
		}
	}

	s.status = flows.SessionStatusActive

	// off to the races again...
	return s.continueUntilWait(waitingRun, destination, step, []flows.CallerEvent{})
}

// finds the next destination in a run that may have been waiting or a parent paused for a child subflow
func (s *session) findResumeDestination(run flows.FlowRun) (flows.NodeUUID, error) {
	step, node, err := run.PathLocation()
	if err != nil {
		return noDestination, err
	}

	// see if this node can now pick a destination
	step, destination, err := s.pickNodeExit(run, node, step)
	if err != nil {
		return noDestination, err
	}

	return destination, nil
}

// the main flow execution loop
func (s *session) continueUntilWait(currentRun flows.FlowRun, destination flows.NodeUUID, step flows.Step, callerEvents []flows.CallerEvent) (err error) {
	for {
		// if we have a flow trigger handle that first to find our destination in the new flow
		if s.pushedFlow != nil {
			// create a new run for it
			flow := s.pushedFlow.flow
			currentRun = runs.NewRun(s, s.pushedFlow.flow, s.contact, currentRun)
			s.addRun(currentRun)
			s.flowStack.push(flow)

			// our destination is the first node in that flow... if such a node exists
			if len(flow.Nodes()) > 0 {
				destination = flow.Nodes()[0].UUID()
			} else {
				destination = noDestination
			}

			// clear the trigger
			s.pushedFlow = nil
		}

		// if we have no destination then we're done with the current run which may have completed, expired or errored
		if destination == noDestination {
			if currentRun.ExitedOn() == nil {
				currentRun.Exit(flows.RunStatusCompleted)
			}

			// switch back our parent run
			if currentRun.ParentInSession() != nil {
				childRun := currentRun
				currentRun = currentRun.ParentInSession()
				s.flowStack.pop()

				// as long as we didn't error, we can try to resume it
				if childRun.Status() != flows.RunStatusErrored {
					if destination, err = s.findResumeDestination(currentRun); err != nil {
						currentRun.AddFatalError(step, nil, fmt.Errorf("can't resume run as node no longer exists"))
					}
				} else {
					// if we did error then that needs to bubble back up through the run hierarchy
					step, _, _ := currentRun.PathLocation()
					currentRun.AddFatalError(step, nil, fmt.Errorf("child run for flow '%s' ended in error, ending execution", childRun.Flow().UUID()))
				}

			} else {
				// If we have no destination and no parent, then the whole session is done. A run error bubbles up the session status.
				if currentRun.Status() == flows.RunStatusErrored {
					s.status = flows.SessionStatusErrored
				} else {
					s.status = flows.SessionStatusCompleted
				}

				// return to caller
				return nil
			}
		}

		// if we now have a destination, go there
		if destination != noDestination {
			if s.flowStack.hasVisited(destination) {
				// this is a loop, we log it and stop execution
				currentRun.AddFatalError(step, nil, fmt.Errorf("flow loop detected, stopping execution before entering '%s'", destination))
				destination = noDestination
			} else {
				node := currentRun.Flow().GetNode(destination)
				if node == nil {
					return fmt.Errorf("unable to find destination node %s in flow %s", destination, currentRun.Flow().UUID())
				}

				step, destination, err = s.visitNode(currentRun, node, callerEvents)
				if err != nil {
					return err
				}

				// if we hit a wait, also return to the caller
				if s.status == flows.SessionStatusWaiting {
					return nil
				}

				// mark this node as visited to prevent loops
				s.flowStack.visit(node.UUID())

				// only pass our caller events to the first node as it is responsible for handling them
				callerEvents = nil
			}
		}
	}
}

// visits the given node, creating a step in our current run path
func (s *session) visitNode(run flows.FlowRun, node flows.Node, callerEvents []flows.CallerEvent) (flows.Step, flows.NodeUUID, error) {
	step := run.CreateStep(node)

	// apply and add any caller events to this new step
	for _, event := range callerEvents {
		if err := event.Apply(run); err != nil {
			return nil, noDestination, err
		}
		run.AddEvent(step, nil, event)
	}

	// execute our node's actions
	if node.Actions() != nil {
		for _, action := range node.Actions() {
			eventLog := actions.NewEventLog()

			if err := action.Execute(run, step, eventLog); err != nil {
				return nil, noDestination, fmt.Errorf("error executing action[type=%s,uuid=%s]: %s", action.Type(), action.UUID(), err)
			}

			if log.GetLevel() >= log.DebugLevel {
				actionEnvelope, _ := utils.EnvelopeFromTyped(action)
				actionJSON, _ := json.Marshal(actionEnvelope)
				log.WithField("action_type", action.Type()).WithField("payload", string(actionJSON)).WithField("run", run.UUID()).Debug("action executed")
			}

			// add any events that the action generated
			for _, event := range eventLog.Events() {
				run.AddEvent(step, action, event)
			}

			// check if this action has errored the run
			if run.Status() == flows.RunStatusErrored {
				return step, noDestination, nil
			}
		}
	}

	// a start flow action may have triggered a subflow in which case we're done on this node for now
	// and it will be resumed when the subflow finishes
	if s.pushedFlow != nil {
		return step, noDestination, nil
	}

	// if our node has a wait before its router, we hand back to the caller
	wait := node.Wait()
	if wait != nil {
		wait.Begin(run, step)

		run.SetStatus(flows.RunStatusWaiting)
		s.wait = wait
		s.status = flows.SessionStatusWaiting

		return step, noDestination, nil
	}

	// use our node's router to determine where to go next
	return s.pickNodeExit(run, node, step)
}

// picks the exit to use on the given node
func (s *session) pickNodeExit(run flows.FlowRun, node flows.Node, step flows.Step) (flows.Step, flows.NodeUUID, error) {
	var err error

	var operand *string
	route := flows.NoRoute
	router := node.Router()

	// we have a router, have it determine our exit
	var exitUUID flows.ExitUUID
	if router != nil {
		if operand, route, err = router.PickRoute(run, node.Exits(), step); err != nil {
			return nil, noDestination, err
		}
		exitUUID = route.Exit()
	} else if len(node.Exits()) > 0 {
		// no router, pick our first exit if we have one
		exitUUID = node.Exits()[0].UUID()
	}

	step.Leave(exitUUID)

	// look up our actual exit and localized name
	var exit flows.Exit
	var localizedExitName string

	if exitUUID != "" {
		// find our exit
		for _, e := range node.Exits() {
			if e.UUID() == exitUUID {
				localizedName := run.GetText(utils.UUID(exitUUID), "name", e.Name())
				if localizedName != e.Name() {
					localizedExitName = localizedName
				}
				exit = e
				break
			}
		}
		if exit == nil {
			return nil, noDestination, fmt.Errorf("unable to find exit with uuid '%s'", exitUUID)
		}
	}

	// no exit? return no destination
	if exit == nil {
		return step, noDestination, nil
	}

	// save our results if appropriate
	if router != nil && router.ResultName() != "" {
		var extraJSON json.RawMessage
		if route.Extra() != nil {
			extraJSON, _ = json.Marshal(route.Extra())
		}
		resultName := router.ResultName()
		resultValue := route.Match()
		resultCategory := exit.Name()
		run.Results().Save(resultName, resultValue, resultCategory, localizedExitName, step.NodeUUID(), operand, extraJSON, utils.Now())
		event := events.NewRunResultChangedEvent(resultName, resultValue, resultCategory, localizedExitName, operand, extraJSON)
		run.AddEvent(step, nil, event)
	}

	return step, exit.DestinationNodeUUID(), nil
}

const noDestination = flows.NodeUUID("")

func (s *session) validateCallerEvents(events []flows.CallerEvent) error {
	for _, event := range events {
		if err := event.Validate(s.assets); err != nil {
			return fmt.Errorf("validation failed for event[type=%s]: %s", event.Type(), err)
		}
	}
	return nil
}

//------------------------------------------------------------------------------------------
// JSON Encoding / Decoding
//------------------------------------------------------------------------------------------

type sessionEnvelope struct {
	Environment json.RawMessage      `json:"environment"`
	Trigger     *utils.TypedEnvelope `json:"trigger"`
	Contact     *json.RawMessage     `json:"contact,omitempty"`
	Runs        []json.RawMessage    `json:"runs"`
	Status      flows.SessionStatus  `json:"status"`
	Wait        *utils.TypedEnvelope `json:"wait,omitempty"`
}

// ReadSession decodes a session from the passed in JSON
func ReadSession(assets flows.SessionAssets, engineConfig flows.EngineConfig, httpClient *utils.HTTPClient, data json.RawMessage) (flows.Session, error) {
	var envelope sessionEnvelope
	var err error

	if err = utils.UnmarshalAndValidate(data, &envelope); err != nil {
		return nil, fmt.Errorf("unable to read session: %s", err)
	}

	s := NewSession(assets, engineConfig, httpClient).(*session)
	s.status = envelope.Status

	// read our environment
	s.env, err = utils.ReadEnvironment(envelope.Environment)
	if err != nil {
		return nil, fmt.Errorf("unable to read environment: %s", err)
	}

	// read our trigger
	if envelope.Trigger != nil {
		if s.trigger, err = triggers.ReadTrigger(s, envelope.Trigger); err != nil {
			return nil, fmt.Errorf("unable to read trigger[type=%s]: %s", envelope.Trigger.Type, err)
		}
	}

	// read our contact
	if envelope.Contact != nil {
		if s.contact, err = flows.ReadContact(s.Assets(), *envelope.Contact); err != nil {
			return nil, fmt.Errorf("unable to read contact: %s", err)
		}
	}

	// read each of our runs
	for i := range envelope.Runs {
		run, err := runs.ReadRun(s, envelope.Runs[i])
		if err != nil {
			return nil, fmt.Errorf("unable to read run %d: %s", i, err)
		}
		s.addRun(run)
	}

	// and our wait
	if envelope.Wait != nil {
		s.wait, err = waits.ReadWait(envelope.Wait)
		if err != nil {
			return nil, fmt.Errorf("unable to read wait[type=%s]: %s", envelope.Wait.Type, err)
		}
	}

	// perform some structural validation
	if s.status == flows.SessionStatusWaiting && s.wait == nil {
		return nil, fmt.Errorf("session has status of \"waiting\" but no wait object")
	}

	return s, nil
}

// MarshalJSON marshals this session into JSON
func (s *session) MarshalJSON() ([]byte, error) {
	var envelope sessionEnvelope
	var err error

	envelope.Status = s.status

	if envelope.Environment, err = json.Marshal(s.env); err != nil {
		return nil, err
	}
	if s.contact != nil {
		var contactJSON json.RawMessage
		contactJSON, err = json.Marshal(s.contact)
		if err != nil {
			return nil, err
		}
		envelope.Contact = &contactJSON
	}
	if s.trigger != nil {
		if envelope.Trigger, err = utils.EnvelopeFromTyped(s.trigger); err != nil {
			return nil, err
		}
	}
	if s.wait != nil {
		if envelope.Wait, err = utils.EnvelopeFromTyped(s.wait); err != nil {
			return nil, err
		}
	}
	envelope.Runs = make([]json.RawMessage, len(s.runs))
	for i := range s.runs {
		envelope.Runs[i], err = json.Marshal(s.runs[i])
		if err != nil {
			return nil, err
		}
	}

	return json.Marshal(envelope)
}
