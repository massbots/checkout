package paymaster

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"go.massbots.xyz/checkout"
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
		BaseURL   string
		AuthToken string
	}

	Amount struct {
		Value    string `json:"value"`
		Currency string `json:"currency"`
	}

	Protocol struct {
		ReturnURL   string `json:"returnUrl"`
		CallbackURL string `json:"callbackUrl"`
	}

	Invoice struct {
		Description string            `json:"description"`
		OrderNumber string            `json:"orderNo"`
		Expires     time.Time         `json:"expires"`
		Params      checkout.Metadata `json:"params"`
	}

	Request struct {
		MerchantID    string   `json:"merchantId"`
		TestMode      bool     `json:"testMode"`
		Invoice       Invoice  `json:"invoice"`
		Amount        Amount   `json:"amount"`
		Protocol      Protocol `json:"protocol"`
		PaymentMethod string   `json:"paymentMethod"`

		Tokenization struct {
			Type        string `json:"type"`
			Purpose     string `json:"purpose"`
			CallbackURL string `json:"callbackUrl"`
		} `json:"tokenization"`

		Customer struct {
			Email   string `json:"email"`
			Phone   string `json:"phone"`
			IP      string `json:"ip"`
			Account string `json:"account"`
		} `json:"customer"`

		Receipt struct {
			Client Client       `json:"client"`
			Items  ReceiptItems `json:"items"`
		} `json:"receipt"`
	}

	Payment struct {
		ID            string    `json:"id"`
		CreatedAt     time.Time `json:"created"`
		TestMode      bool      `json:"testMode"`
		Status        string    `json:"status"`
		MerchantID    string    `json:"merchantId"`
		Invoice       Invoice   `json:"invoice"`
		PaymentMethod string    `json:"paymentMethod"`
		Amount        Amount    `json:"amount"`

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
		MerchantID:    payment.MerchantID,
		PaymentMethod: payment.PaymentMethod,
		Invoice:       Invoice{Description: payment.Comment},
		Amount:        Amount{Value: payment.Amount, Currency: payment.Currency},
		Protocol:      Protocol{ReturnURL: payment.SuccessURL, CallbackURL: payment.CallbackURL},
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodPost, BaseURL+"/invoices", bytes.NewReader(data))
	req.Header.Set("Authorization", c.AuthToken)
	req.Header.Set("Idempotency-Key", payment.ID)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}

	dec := json.NewDecoder(resp.Body)
	defer resp.Body.Close()

	var result struct {
		ID  string `json:"paymentId"`
		URL string `json:"url"`
	}

	return result.URL, dec.Decode(&result)
}

func (c Checkout) Webhook(callback checkout.Callback) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var p Payment
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			log.Println("checkout/paymaster:", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		payment := checkout.Payment{
			Checkout: "paymaster",
			ID:       p.ID,
			Amount:   p.Amount.Value,
			Currency: p.Amount.Currency,
			Comment:  p.Invoice.Description,
			Status:   statuses[p.Status],
			Profit:   p.Amount.Value,
			PaidAt:   p.CreatedAt,
			V:        p,
		}

		if err := callback(payment); err != nil {
			log.Println("checkout/paymaster:", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	})
}
