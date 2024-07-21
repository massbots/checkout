package yookassa

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"go.massbots.xyz/checkout"
)

const BaseURL = "https://api.yookassa.ru/v3/payments"

type (
	// Checkout implements checkout.Checkout.
	Checkout struct {
		ShopID string
		APIKey string
	}

	Amount struct {
		Value    string `json:"value"`
		Currency string `json:"currency"`
	}

	Confirmation struct {
		Type      string `json:"type"`
		ReturnURL string `json:"return_url"`
	}

	Request struct {
		Description  string       `json:"description"`
		Amount       Amount       `json:"amount"`
		Confirmation Confirmation `json:"confirmation"`
		Capture      bool         `json:"capture"`
	}

	Payment struct {
		ID          string            `json:"id"`
		Status      string            `json:"status"`
		Test        bool              `json:"test"`
		Paid        bool              `json:"paid"`
		Amount      Amount            `json:"amount"`
		Income      Amount            `json:"income_amount"`
		Created     time.Time         `json:"created_at"`
		Captured    time.Time         `json:"captured_at"`
		Expires     time.Time         `json:"expires_at"`
		Description string            `json:"description"`
		Metadata    checkout.Metadata `json:"metadata"`

		Recipient struct {
			AccountID string `json:"account_id"`
			GatewayID string `json:"gateway_id"`
		} `json:"recipient"`

		Confirmation struct {
			URL string `json:"confirmation_url"`
		} `json:"confirmation"`
	}

	Event struct {
		Type   string  `json:"type"`
		Name   string  `json:"event"`
		Object Payment `json:"object"`
	}
)

// From returns the original payment structure.
func From(payment checkout.Payment) Payment {
	p, _ := payment.V.(Payment)
	return p
}

func idempotenceKey() (string, error) {
	key, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	return key.String(), nil
}

func (c Checkout) Request(payment checkout.Payment) (string, error) {
	data, err := json.Marshal(Request{
		Description:  payment.Comment,
		Amount:       Amount{Value: payment.Amount, Currency: payment.Currency},
		Confirmation: Confirmation{Type: "redirect", ReturnURL: payment.SuccessURL},
		Capture:      true,
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodPost, BaseURL, bytes.NewReader(data))
	if err != nil {
		return "", err
	}

	req.SetBasicAuth(c.ShopID, c.APIKey)
	req.Header.Set("Idempotence-Key", payment.ID)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result Payment
	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.Confirmation.URL, nil
}

var statuses = map[string]int{
	"waiting_for_capture": checkout.StatusWaiting,
	"succeeded":           checkout.StatusPaid,
	"canceled":            checkout.StatusRejected,
}

func (c Checkout) Webhook(callback checkout.Callback) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var event Event
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			log.Println("checkout/yookassa:", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		payment := checkout.Payment{
			Checkout: "yookassa",
			ID:       event.Object.ID,
			Amount:   event.Object.Amount.Value,
			Currency: event.Object.Amount.Currency,
			Comment:  event.Object.Description,
			Status:   statuses[event.Object.Status],
			Profit:   event.Object.Income.Value,
			PaidAt:   event.Object.Captured,
			V:        event.Object,
		}

		if err := callback(payment); err != nil {
			log.Println("checkout/yookassa:", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	})
}
