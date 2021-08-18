# Checkout
> `go get go.massbots.xyz/checkout`

The goal of `checkout` package is to unite popular acquiring providers for quick payment integration. We mostly use it in our Telegram bots.

## Supported providers

Feel free to request a missing provider by creating an issue, or adding one you'd like to integrate by making a pull request.

- [Qiwi](https://p2p.qiwi.com)
- [YooMoney](https://yoomoney.ru)
- [YooKassa](https://yookassa.ru)
- [Anypay](https://anypay.io)
- [Enotio](https://enot.io)

## Usage example

```go
package main

import (
	"net/http"
	"os"

	"go.massbots.xyz/checkout"
	"go.massbots.xyz/checkout/yookassa"
)

func main() {
	co := &yookassa.Checkout{
		ShopID: os.Getenv("YOO_SHOP_ID"),
		APIKey: os.Getenv("YOO_API_KEY"),
	}

	// Generate a link for the user
	url, err := co.Request(checkout.Payment{
		ID:       "1",
		Amount:   "100.00",
		Currency: checkout.RUB,
		Metadata: checkout.Metadata{...},
	})

	// Process incoming updates
	http.Handle("/process", co.Webhook(callback))
	go http.ListenAndServe(":8080", nil)

	// Do other stuff...
}

func callback(p checkout.Payment) error {
	// Payment is successful!

	// New fields to process:
	// p.Status
	// p.Profit
	// p.PaidAt
	// p.Metadata

	// In case you need original full structure
	pp := yookassa.From(p)
}
```