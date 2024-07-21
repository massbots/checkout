package enotio

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

const BaseURL = "https://enot.io/pay?"

// Checkout implements checkout.Checkout.
type Checkout struct {
	MerchantID string
	APIKey1    string
	APIKey2    string
}

func (c Checkout) encodeMetadata(md checkout.Metadata) string {
	var a []string
	for k, v := range md {
		a = append(a, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(a, ",")
}

func (c Checkout) decodeMetadata(s string) checkout.Metadata {
	md := make(checkout.Metadata)
	for _, a := range strings.Split(s, ",") {
		kv := strings.Split(a, "=")
		md[kv[0]] = strings.Join(kv[1:], "")
	}
	return md
}

func (c Checkout) Request(payment checkout.Payment) (string, error) {
	params := url.Values{}
	params.Set("m", c.MerchantID)
	params.Set("o", payment.ID)
	params.Set("oa", payment.Amount)
	params.Set("cf", c.encodeMetadata(payment.Metadata))

	a := strings.Join([]string{
		c.MerchantID,
		payment.Amount,
		c.APIKey1,
		payment.ID,
	}, ":")

	hash := md5.Sum([]byte(a))
	params.Set("s", hex.EncodeToString(hash[:]))
	return BaseURL + params.Encode(), nil
}

func (c Checkout) Webhook(callback checkout.Callback) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			log.Println("checkout/enotio:", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		a := strings.Join([]string{
			c.MerchantID,
			r.FormValue("amount"),
			c.APIKey2,
			r.FormValue("merchant_id"),
		}, ":")

		hash := md5.Sum([]byte(a))
		if r.FormValue("sign_2") != hex.EncodeToString(hash[:]) {
			log.Println("checkout/enotio: bad signature")
			w.WriteHeader(http.StatusForbidden)
			return
		}

		custom, _ := url.QueryUnescape(r.FormValue("custom_field"))
		metadata := c.decodeMetadata(custom)

		payment := checkout.Payment{
			Checkout: "enotio",
			ID:       r.FormValue("merchant_id"),
			Currency: r.FormValue("currency"),
			Amount:   r.FormValue("amount"),
			Metadata: metadata,
			Status:   checkout.StatusPaid,
			Profit:   r.FormValue("credited"),
			PaidAt:   time.Now(),
		}

		if err := callback(payment); err != nil {
			log.Println("checkout/enotio:", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	})
}
