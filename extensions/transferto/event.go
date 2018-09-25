package transferto

import (
	"github.com/nyaruka/goflow/flows"
	"github.com/nyaruka/goflow/flows/events"

	"github.com/shopspring/decimal"
)

func init() {
	events.RegisterType(TypeAirtimeTransfered, func() flows.Event { return &AirtimeTransferredEvent{} })
}

// TypeAirtimeTransfered is the type of our airtime transferred event
const TypeAirtimeTransfered string = "airtime_transferred"

// AirtimeTransferredEvent events are created when airtime has been transferred to the contact
//
//   {
//     "type": "airtime_transferred",
//     "created_on": "2006-01-02T15:04:05Z",
//     "currency": "RWF",
//     "amount": 100,
//     "status": "success"
//   }
//
// @event airtime_transferred
type AirtimeTransferredEvent struct {
	events.BaseEvent

	Currency string          `json:"currency"`
	Amount   decimal.Decimal `json:"amount"`
	Status   string          `json:"status"`
}

// NewAirtimeTransferredEvent creates a new airtime transferred event
func NewAirtimeTransferredEvent(currency string, amount decimal.Decimal) *AirtimeTransferredEvent {
	return &AirtimeTransferredEvent{
		BaseEvent: events.NewBaseEvent(),
		Currency:  currency,
		Amount:    amount,
		Status:    "success",
	}
}

// NewFailedAirtimeTransferredEvent creates a new failed airtime transferred event
func NewFailedAirtimeTransferredEvent() *AirtimeTransferredEvent {
	return &AirtimeTransferredEvent{
		BaseEvent: events.NewBaseEvent(),
		Amount:    decimal.Zero,
		Status:    "failed",
	}
}

// Type returns the type of this event
func (e *AirtimeTransferredEvent) Type() string { return TypeAirtimeTransfered }
