package anypay

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.massbots.xyz/checkout"
)

const BaseURL = "https://anypay.io/merchant?"

// Checkout implements checkout.Checkout.
type Checkout struct {
	MerchantID string
	APIKey     string
}

func (c Checkout) Request(payment checkout.Payment) (string, error) {
	params := url.Values{}
	params.Set("merchant_id", c.MerchantID)
	params.Set("pay_id", payment.ID)
	params.Set("amount", payment.Amount)
	params.Set("currency", payment.Currency)

	for k, v := range payment.Metadata {
		params.Set(k, fmt.Sprint(v))
	}

	a := strings.Join([]string{
		payment.Currency,
		payment.Amount,
		c.APIKey,
		c.MerchantID,
		payment.ID,
	}, ":")

	hash := md5.Sum([]byte(a))
	params.Set("sign", hex.EncodeToString(hash[:]))
	return BaseURL + params.Encode(), nil
}

var (
	timeLayout = "02.01.2006 15:04:05"
	timeLoc, _ = time.LoadLocation("Europe/Moscow")
)

func (c Checkout) Webhook(callback checkout.Callback) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			log.Println("checkout/anypay:", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		paidAt, err := time.ParseInLocation(timeLayout, r.FormValue("pay_date"), timeLoc)
		if err != nil {
			log.Println("checkout/anypay:", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		payment := checkout.Payment{
			Checkout: "anypay",
			ID:       r.FormValue("pay_id"),
			Amount:   r.FormValue("amount"),
			Currency: r.FormValue("currency"),
			Metadata: make(checkout.Metadata),
			Status:   checkout.StatusPaid,
			Profit:   r.FormValue("profit"),
			PaidAt:   paidAt.UTC(),
		}

		a := strings.Join([]string{
			c.MerchantID,
			payment.Amount,
			payment.ID,
			c.APIKey,
		}, ":")

		hash := md5.Sum([]byte(a))
		if r.FormValue("sign") != hex.EncodeToString(hash[:]) {
			log.Println("checkout/anypay: bad signature")
			w.WriteHeader(http.StatusForbidden)
			return
		}

		for k, v := range r.Form {
			payment.Metadata[k] = v
		}

		if err := callback(payment); err != nil {
			log.Println("checkout/anypay:", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	})
}
