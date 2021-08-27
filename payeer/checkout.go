package payeer

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.massbots.xyz/checkout"
)

const BaseURL = "https://payeer.com/merchant/?"

// Checkout implements checkout.Checkout.
type Checkout struct {
	MerchantID string
	APIKey     string
}

// Request implements Checkout.Request. Does not support Metadata.
func (c Checkout) Request(payment checkout.Payment) (string, error) {
	params := url.Values{}
	params.Set("m_shop", c.MerchantID)
	params.Set("m_orderid", payment.ID)
	params.Set("m_amount", payment.Amount)
	params.Set("m_curr", payment.Currency)
	desc := base64.StdEncoding.EncodeToString([]byte(payment.Comment))
	params.Set("m_desc", desc)

	a := strings.Join([]string{
		c.MerchantID,
		payment.ID,
		payment.Amount,
		payment.Currency,
		desc,
		c.APIKey,
	}, ":")

	hash := sha256.Sum256([]byte(a))
	params.Set("m_sign", strings.ToUpper(hex.EncodeToString(hash[:])))
	return BaseURL + params.Encode(), nil
}

var (
	timeLayout = "02.01.2006 15:04:05"
	timeLoc, _ = time.LoadLocation("Europe/Moscow")
)

var statuses = map[string]int{
	"success": checkout.StatusPaid,
}

// Webhook implements Checkout.Webhook.
//
// Does not support Profit. Amount will be equal to profit
// if commission is on the buyer.
//
func (c Checkout) Webhook(callback checkout.Callback) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			log.Println("checkout/payeer:", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		a := []string{
			r.FormValue("m_operation_id"),
			r.FormValue("m_operation_ps"),
			r.FormValue("m_operation_date"),
			r.FormValue("m_operation_pay_date"),
			r.FormValue("m_shop"),
			r.FormValue("m_orderid"),
			r.FormValue("m_amount"),
			r.FormValue("m_curr"),
			r.FormValue("m_desc"),
			r.FormValue("m_status"),
		}
		if r.FormValue("m_params") != "" {
			a = append(a, r.FormValue("m_params"))
		}
		a = append(a, c.APIKey)

		hash := sha256.Sum256([]byte(strings.Join(a, ":")))
		if r.FormValue("m_sign") != strings.ToUpper(hex.EncodeToString(hash[:])) {
			log.Println("checkout/payeer: bad request")
			w.WriteHeader(http.StatusForbidden)
			return
		}

		paidAt, err := time.ParseInLocation(timeLayout, r.FormValue("m_operation_pay_date"), timeLoc)
		if err != nil {
			log.Println("checkout/payeer:", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		comment, _ := base64.StdEncoding.DecodeString(r.FormValue("m_desc"))

		payment := checkout.Payment{
			Checkout: "payeer",
			ID:       r.FormValue("m_orderid"),
			Currency: r.FormValue("m_curr"),
			Comment:  string(comment),
			Status:   statuses[r.FormValue("m_status")],
			Amount:   r.FormValue("m_amount"),
			Profit:   r.FormValue("summa_out"),
			PaidAt:   paidAt,
		}

		if err := callback(payment); err != nil {
			log.Println("checkout/payeer:", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(r.FormValue("m_orderid") + "|error"))
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(r.FormValue("m_orderid") + "|success"))
	})
}
