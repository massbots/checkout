package paymaster

import (
	"bytes"
	"encoding/json"
	"go.massbots.xyz/checkout"
	"log"
	"net/http"
	"time"
)

var BaseURL = "https://paymaster.ru/api/v2"

var statuses = map[string]int{
	"Pending":   checkout.StatusWaiting,
	"Settled":   checkout.StatusPaid,
	"Cancelled": checkout.StatusRejected,
}

// Checkout implements checkout.Checkout.
type (
	Checkout struct {
		BaseURL             string
		AuthorizationBearer string
	}

	Amount struct {
		Value    string `json:"value"`
		Currency string `json:"currency"`
	}

	Invoice struct {
		Description string            `json:"description"`
		OrderNumber string            `json:"orderNo"`
		Expires     time.Time         `json:"expires"`
		Params      checkout.Metadata `json:"params"`
	}

	Request struct {
		MerchantID   string `json:"merchantId"`
		TestMode     bool   `json:"testMode"`
		Tokenization struct {
			Type        string `json:"type"`
			Purpose     string `json:"purpose"`
			CallbackURL string `json:"callbackUrl"`
		} `json:"tokenization"`
		Invoice       Invoice `json:"invoice"`
		Amount        Amount  `json:"amount"`
		PaymentMethod string  `json:"paymentMethod"`
		Protocol      struct {
			ReturnURL   string `json:"returnUrl"`
			CallbackURL string `json:"callbackUrl"`
		}
	}

	Payment struct {
		ID            string    `json:"id"`
		CreatedAt     time.Time `json:"created"`
		TestMode      bool      `json:"testMode"`
		Status        string    `json:"status"`
		MerchantID    string    `json:"merchantId"`
		Invoice       Invoice   `json:"invoice"`
		PaymentMethod string    `json:"paymentMethod"`

		Amount Amount `json:"amount"`

		PaymentData struct {
			PaymentMethod         string `json:"paymentMethod"`
			PaymentInstrumentTile string `json:"paymentInstrumentTile"`
		} `json:"paymentData"`
	}
)

func (c Checkout) Request(payment checkout.Payment) (string, error) {
	if c.BaseURL == "" {
		c.BaseURL = BaseURL
	}

	data, err := json.Marshal(Request{
		MerchantID: payment.MerchantID,
		TestMode:   false,
		Tokenization: struct {
			Type        string `json:"type"`
			Purpose     string `json:"purpose"`
			CallbackURL string `json:"callbackUrl"`
		}{},
		Invoice: Invoice{
			Description: payment.Comment,
		},
		Amount: Amount{
			Value: payment.Amount, Currency: payment.Currency,
		},
		PaymentMethod: payment.PaymentMethod,
		Protocol: struct {
			ReturnURL   string `json:"returnUrl"`
			CallbackURL string `json:"callbackUrl"`
		}{
			ReturnURL:   payment.SuccessURL,
			CallbackURL: payment.CallbackURL,
		},
	})

	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodPost, BaseURL+"/invoices", bytes.NewReader(data))

	req.Header.Set("Authorization", c.AuthorizationBearer)
	req.Header.Set("Idempotency-Key", payment.ID)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		ID  string `json:"paymentId"`
		URL string `json:"url"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.URL, nil
}

func (c Checkout) Webhook(callback checkout.Callback) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var localPayment Payment
		if err := json.NewDecoder(r.Body).Decode(&localPayment); err != nil {
			log.Println("checkout/paymaster:", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		payment := checkout.Payment{
			Checkout: "paymaster",
			ID:       localPayment.ID,
			Amount:   localPayment.Amount.Value,
			Currency: localPayment.Amount.Currency,
			Comment:  localPayment.Invoice.Description,
			Status:   statuses[localPayment.Status],
			Profit:   localPayment.Amount.Value,
			PaidAt:   localPayment.CreatedAt,
			V:        localPayment,
		}

		if err := callback(payment); err != nil {
			log.Println("checkout/paymaster:", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	})
}
