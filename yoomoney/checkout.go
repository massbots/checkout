package yoomoney

import (
	"crypto/sha1"
	"encoding/hex"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"go.massbots.xyz/checkout"
)

const BaseURL = "https://yoomoney.ru/quickpay/confirm.xml?"

// Payment types.
const (
	PC = "PC" // wallet
	AC = "AC" // card
	MC = "MC" // mobile
)

type (
	// Checkout implements checkout.Checkout.
	Checkout struct {
		Receiver  string
		SecretKey string
	}
)

// WithCommission returns the given amount summed with the corresponding
// to the payment type commission.
func WithCommission(pt, amount string) string {
	a, _ := decimal.NewFromString(amount)

	switch pt {
	case PC:
		const commission float64 = 0.005 / 1.005
		return a.Add(a.Mul(decimal.NewFromFloat(commission))).StringFixed(2)
	case AC:
		const commission float64 = 1 - 0.02
		return a.Div(decimal.NewFromFloat(commission)).StringFixed(2)
	}

	return amount
}

// Request implements Checkout.Request. Does not support Metadata.
func (c Checkout) Request(payment checkout.Payment) (string, error) {
	params := url.Values{}
	params.Set("receiver", c.Receiver)
	params.Set("quickpay-form", "shop")
	params.Set("paymentType", payment.Type)
	params.Set("targets", payment.Target)
	params.Set("sum", payment.Amount)
	params.Set("comment", payment.Comment)
	params.Set("label", payment.ID)
	params.Set("successURL", payment.SuccessURL)

	return BaseURL + params.Encode(), nil
}

var (
	timeLayout = "2006-01-02T15:04:05Z"
	timeLoc, _ = time.LoadLocation("Europe/Moscow")
)

func (c Checkout) Webhook(callback checkout.Callback) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			log.Println("checkout/yoomoney:", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		paidAt, err := time.ParseInLocation(timeLayout, r.FormValue("datetime"), timeLoc)
		if err != nil {
			log.Println("checkout/yoomoney:", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		payment := checkout.Payment{
			Checkout: "yoomoney",
			ID:       r.FormValue("label"),
			Amount:   r.FormValue("withdraw_amount"),
			Currency: r.FormValue("currency"),
			Status:   checkout.StatusPaid,
			Profit:   r.FormValue("amount"),
			PaidAt:   paidAt.UTC(),
		}

		a := strings.Join([]string{
			r.FormValue("notification_type"),
			r.FormValue("operation_id"),
			payment.Profit,
			payment.Currency,
			r.FormValue("datetime"),
			r.FormValue("sender"),
			r.FormValue("codepro"),
			c.SecretKey,
			r.FormValue("label"),
		}, "&")

		hash := sha1.Sum([]byte(a))
		if r.FormValue("sha1_hash") != hex.EncodeToString(hash[:]) {
			log.Println("checkout/yoomoney: bad signature")
			w.WriteHeader(http.StatusForbidden)
			return
		}

		if err := callback(payment); err != nil {
			log.Println("checkout/yoomoney:", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	})
}
