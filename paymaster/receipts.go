package paymaster

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"time"
)

type (
	Receipt struct {
		ID        string     `json:"id,omitempty"`
		CreatedAt *time.Time `json:"created,omitempty"`
		PaymentID string     `json:"paymentId,omitempty"`
		Amount    *Amount    `json:"amount,omitempty"`
		Type      string     `json:"type,omitempty"`
		Status    string     `json:"status,omitempty"`

		Client *ReceiptClient `json:"client,omitempty"` // request
		Items  []*ReceiptItem `json:"items,omitempty"`  // request
	}

	ReceiptClient struct {
		Email string `json:"email,omitempty"`
		Phone string `json:"phone,omitempty"`
		Name  string `json:"name,omitempty"`
		INN   string `json:"INN,omitempty"`
	}

	ReceiptItem struct {
		Name           string           `json:"name,omitempty"`
		Quantity       string           `json:"quantity,omitempty"` // decimal
		Price          string           `json:"price,omitempty"`    // decimal
		Excise         string           `json:"excise,omitempty"`   // decimal
		Measure        string           `json:"measure,omitempty"`
		VatType        string           `json:"vatType,omitempty"`
		PaymentSubject string           `json:"paymentSubject,omitempty"`
		PaymentMethod  string           `json:"paymentMethod,omitempty"`
		Product        *ReceiptProduct  `json:"product,omitempty"`
		Marking        *ReceiptMarking  `json:"marking,omitempty"`
		Supplier       *ReceiptSupplier `json:"supplier,omitempty"`
	}

	ReceiptProduct struct {
		Country     string `json:"country,omitempty"`
		Declaration string `json:"declaration,omitempty"`
	}

	ReceiptMarking struct {
		Code      string           `json:"code,omitempty"`
		AgentType string           `json:"agentType,omitempty"`
		Quantity  *ReceiptQuantity `json:"quantity,omitempty"`
	}

	ReceiptQuantity struct {
		Numerator   int `json:"numerator,omitempty"`
		Denominator int `json:"denominator,omitempty"`
	}

	ReceiptSupplier struct {
		Name  string `json:"name,omitempty"`
		INN   string `json:"INN,omitempty"`
		Phone string `json:"phone,omitempty"`
	}
)

func (c Checkout) CreateReceipt(r Receipt) (*Receipt, error) {
	end := c.BaseURL + "/receipts"

	data, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, end, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", c.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	return &r, json.NewDecoder(resp.Body).Decode(&r)
}

func (c Checkout) ReceiptByID(id string) (r *Receipt, err error) {
	end := c.BaseURL + "/receipts/" + id

	req, err := http.NewRequest(http.MethodPost, end, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", c.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	return r, json.NewDecoder(resp.Body).Decode(r)
}

func (c Checkout) Receipts(paymentID string) (r []Receipt, err error) {
	params := url.Values{}
	params.Set("paymentId", paymentID)

	end := c.BaseURL + "/receipts" + params.Encode()

	req, err := http.NewRequest(http.MethodPost, end, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", c.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	return r, json.NewDecoder(resp.Body).Decode(&r)
}
