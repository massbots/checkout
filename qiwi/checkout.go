package qiwi

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.massbots.xyz/checkout"
)

const BaseURL = "https://oplata.qiwi.com/create?"

type (
	// Checkout implements checkout.Checkout.
	Checkout struct {
		BaseURL   string
		PublicKey string
		SecretKey string
	}

	Payment struct {
		SiteID             string            `json:"siteId"`
		BillID             string            `json:"billId"`
		CustomFields       checkout.Metadata `json:"customFields"`
		Comment            string            `json:"comment"`
		CreationDateTime   string            `json:"creationDateTime"`
		ExpirationDateTime string            `json:"expirationDateTime"`

		Amount struct {
			Value    string `json:"value"`
			Currency string `json:"currency"`
		} `json:"amount"`

		Status struct {
			Value           string `json:"value"`
			ChangedDateTime string `json:"changedDateTime"`
		} `json:"status"`

		Customer struct {
			Phone   string `json:"phone"`
			Email   string `json:"email"`
			Account string `json:"account"`
		} `json:"customer"`
	}
)

// From returns the original payment structure.
func From(payment checkout.Payment) Payment {
	p, _ := payment.V.(Payment)
	return p
}

func (c Checkout) Request(payment checkout.Payment) (string, error) {
	if c.BaseURL == "" {
		c.BaseURL = BaseURL
	}

	params := url.Values{}
	params.Set("publicKey", c.PublicKey)
	params.Set("billId", payment.ID)
	params.Set("amount", payment.Amount)
	params.Set("comment", payment.Comment)
	params.Set("successUrl", payment.SuccessURL)

	expDate := payment.ExpirationDate.Format("2006-01-02T15:04:05-07:00")
	params.Set("expirationDateTime", expDate)

	for k, v := range payment.Metadata {
		params.Set("customFields["+k+"]", fmt.Sprint(v))
	}

	return c.BaseURL + params.Encode(), nil
}

var timeLayout = "2006-01-02T15:04:05-07"

var statuses = map[string]int{
	"WAITING":  checkout.StatusWaiting,
	"PAID":     checkout.StatusPaid,
	"REJECTED": checkout.StatusRejected,
	"EXPIRED":  checkout.StatusExpired,
}

func (c Checkout) Webhook(callback checkout.Callback) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var bill struct {
			Payment Payment `json:"bill"`
		}
		if err := json.NewDecoder(r.Body).Decode(&bill); err != nil {
			log.Println("checkout/qiwi:", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		paidAt, err := time.Parse(timeLayout, bill.Payment.CreationDateTime)
		if err != nil {
			log.Println("checkout/qiwi:", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		payment := checkout.Payment{
			Checkout: "qiwi",
			ID:       bill.Payment.BillID,
			Currency: bill.Payment.Amount.Currency,
			Comment:  bill.Payment.Comment,
			Metadata: bill.Payment.CustomFields,
			Status:   statuses[bill.Payment.Status.Value],
			Profit:   bill.Payment.Amount.Value,
			PaidAt:   paidAt,
			V:        bill.Payment,
		}

		a := strings.Join([]string{
			payment.Currency,
			payment.Profit,
			payment.ID,
			bill.Payment.SiteID,
			bill.Payment.Status.Value,
		}, "|")

		hash := hmac.New(sha256.New, []byte(c.SecretKey))
		hash.Write([]byte(a))

		sign := r.Header.Get("X-Api-Signature-SHA256")
		if sign != hex.EncodeToString(hash.Sum(nil)) {
			log.Println("checkout/qiwi: bad signature")
			w.WriteHeader(http.StatusForbidden)
			return
		}

		if err := callback(payment); err != nil {
			log.Println("checkout/qiwi:", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	})
}
