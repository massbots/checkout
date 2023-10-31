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

// Statuses.
const (
	StatusPaid = 1 + iota
	StatusWaiting
	StatusExpired
	StatusRejected
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
		ID         string
		Amount     string
		Currency   string
		Comment    string
		SuccessURL string
		Metadata   Metadata

		Target         string    // yoomoney only
		Type           string    // yoomoney, paymaster only
		ExpirationDate time.Time // qiwi,paymaster only
		CallbackURL    string    // paymaster only
		PaymentMethod  string    // paymaster only
		Customer       string    // paymaster only

		Checkout string    // in callback only
		Status   int       // in callback only
		Profit   string    // in callback only
		PaidAt   time.Time // in callback only

		// V is a special field set by a checkout implementation. It stores an
		// original payment structure.
		//
		// Example:
		// 		func callback(p checkout.Payment) error {
		//			pp := yookassa.From(p) // yookassa.Payment
		//		}
		//
		V interface{} `json:"-"`
	}

	// Metadata is a set of custom fields necessary to be passed to the payment request.
	Metadata = map[string]interface{}

	// Callback is a function called by a checkout as a result of webhook triggering.
	Callback = func(Payment) error
)
