package paymaster

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"go.massbots.xyz/checkout"
)

const BaseURL = "https://paymaster.ru/api/v2"

var statuses = map[string]int{
	"Pending":   checkout.StatusWaiting,
	"Settled":   checkout.StatusPaid,
	"Cancelled": checkout.StatusRejected,
}

// Checkout implements checkout.Checkout.
type Checkout struct {
	BaseURL    string
	Token      string
	MerchantID string
}

type (
	Request struct {
		MerchantID    string        `json:"merchantId"`
		TestMode      bool          `json:"testMode,omitempty"`
		PaymentMethod string        `json:"paymentMethod,omitempty"`
		Invoice       *Invoice      `json:"invoice,omitempty"`
		Amount        *Amount       `json:"amount,omitempty"`
		Protocol      *Protocol     `json:"protocol,omitempty"`
		Tokenization  *Tokenization `json:"tokenization,omitempty"`
		Receipt       *Receipt      `json:"receipt,omitempty"`
		Customer      *Customer     `json:"customer,omitempty"`
	}

	Amount struct {
		Value    string `json:"value,omitempty"`
		Currency string `json:"currency,omitempty"`
	}

	Protocol struct {
		ReturnURL   string `json:"returnUrl,omitempty"`
		CallbackURL string `json:"callbackUrl,omitempty"`
	}

	Invoice struct {
		Description string            `json:"description,omitempty"`
		OrderNumber string            `json:"orderNo,omitempty"`
		Expires     time.Time         `json:"expires,omitempty"`
		Params      checkout.Metadata `json:"params,omitempty"`
	}

	Tokenization struct {
		Type        string `json:"type,omitempty"`
		Purpose     string `json:"purpose,omitempty"`
		CallbackURL string `json:"callbackUrl,omitempty"`
	}

	Customer struct {
		Email   string `json:"email,omitempty"`
		Phone   string `json:"phone,omitempty"`
		IP      string `json:"ip,omitempty"`
		Account string `json:"account,omitempty"`
	}

	Payment struct {
		ID            string      `json:"id"`
		CreatedAt     time.Time   `json:"created"`
		TestMode      bool        `json:"testMode"`
		Status        string      `json:"status"`
		MerchantID    string      `json:"merchantId"`
		Invoice       Invoice     `json:"invoice"`
		PaymentMethod string      `json:"paymentMethod"`
		Amount        Amount      `json:"amount"`
		PaymentData   PaymentData `json:"paymentData"`
	}

	PaymentData struct {
		PaymentMethod         string `json:"paymentMethod"`
		PaymentInstrumentTile string `json:"paymentInstrumentTile"`
	}
)

func (c Checkout) Raw(end, ik string, v, r any) error {
	end = c.BaseURL + "/" + end

	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, end, bytes.NewReader(data))
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Idempotency-Key", ik)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err = io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var maybeError struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}

	err = json.Unmarshal(data, &maybeError)
	if err == nil && maybeError.Code != "" {
		return fmt.Errorf(
			"checkout/paymaster: %s (%s)",
			maybeError.Code,
			maybeError.Message,
		)
	}

	return json.Unmarshal(data, r)
}

func (c Checkout) Request(p checkout.Payment) (string, error) {
	req := Request{
		MerchantID:    c.MerchantID,
		PaymentMethod: p.PaymentMethod,
		Customer:      &Customer{Account: p.Customer},

		Protocol: &Protocol{
			ReturnURL:   p.SuccessURL,
			CallbackURL: p.CallbackURL,
		},
		Invoice: &Invoice{
			Description: p.Comment,
			Params:      p.Metadata,
			Expires:     p.ExpirationDate, // a must
		},
		Amount: &Amount{
			Value:    p.Amount,
			Currency: p.Currency,
		},
		Tokenization: &Tokenization{
			Type:        p.Type,
			Purpose:     p.Comment,
			CallbackURL: p.CallbackURL,
		},
	}

	var result map[string]string
	return result["url"], c.Raw("invoices", p.ID, req, &result)
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
