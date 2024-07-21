package paymaster

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
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
		PaymentData   *PaymentData  `json:"paymentData,omitempty"`
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
		Expires     *time.Time        `json:"expires,omitempty"`
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

	PaymentData struct {
		Method string  `json:"paymentMethod,omitempty"`
		Token  TokenID `json:"token,omitempty"`
	}

	TokenID struct {
		ID string `json:"id,omitempty"`
	}

	Payment struct {
		ID        int       `json:"id"`
		CreatedAt time.Time `json:"created"`
		Status    string    `json:"status"`
		Invoice   Invoice   `json:"invoice"`
		Amount    Amount    `json:"amount"`
	}
)

func (c Checkout) Raw(end string, r, v any, ik string) error {
	end = c.BaseURL + "/" + end

	data, err := json.Marshal(r)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, end, bytes.NewReader(data))
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")

	if ik != "" {
		req.Header.Set("Idempotency-Key", ik)
	}

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

	return json.Unmarshal(data, v)
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
			Expires:     &p.ExpirationDate, // a must
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
	return result["url"], c.Raw("invoices", req, &result, p.ID)
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
			ID:       strconv.Itoa(p.ID),
			Amount:   p.Amount.Value,
			Currency: p.Amount.Currency,
			Comment:  p.Invoice.Description,
			Status:   statuses[p.Status],
			Profit:   p.Amount.Value,
			PaidAt:   p.CreatedAt,
			Metadata: p.Invoice.Params,
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

func (a *Amount) UnmarshalJSON(b []byte) error {
	var v struct {
		Value    float64 `json:"value"`
		Currency string  `json:"currency"`
	}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}

	a.Value = fmt.Sprint(v.Value)
	a.Currency = v.Currency
	return nil
}
