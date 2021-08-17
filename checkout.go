package checkout

import (
	"net/http"
	"time"
)

// Currencies.
const (
	RUB = "RUB"
	UAH = "UAH"
	USD = "USD"
)

type (
	// Checkout provides two primary operations from a chosen payment acquiring.
	Checkout interface {
		// Request builds up a payment link intended for the end user.
		Request(Payment) (string, error)
		// Webhook returns an http handler that checks a signature and calls the
		// callback for further processing on success.
		Webhook(Callback) http.Handler
	}

	// Payment represents a universal payment object.
	Payment struct {
		ID        string
		AccountID string
		Amount    string
		Currency  string
		Comment   string
		ReturnURL string
		Metadata  Metadata

		Status string    // for callback only
		Profit string    // for callback only
		PaidAt time.Time // for callback only

		// V is a special field set by a checkout implementation. It stores an
		// original payment structure.
		//
		// Example:
		// 		func callback(p checkout.Payment) error {
		//			pp := yookassa.From(p) // yookassa.Payment
		//		}
		//
		V interface{}
	}

	// Metadata is a set of custom fields necessary to be passed to the payment request.
	Metadata = map[string]interface{}

	// Callback is a function called by a checkout as a result of webhook triggering.
	Callback = func(Payment) error
)
