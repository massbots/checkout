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
		ID        string    `json:"id"`
		CreatedAt time.Time `json:"created"`
		PaymentID string    `json:"paymentId"`
		Amount    Amount    `json:"amount"`
		Type      string    `json:"type"`
		Status    string    `json:"status"`

		Client ReceiptClient `json:"client"` // request
		Items  ReceiptItems  `json:"items"`  // request
	}

	ReceiptClient struct {
		Email string `json:"email"`
		Phone string `json:"phone"`
		Name  string `json:"name"`
		INN   string `json:"INN"`
	}

	ReceiptItems struct {
		Name           string `json:"name"`
		Quantity       string `json:"quantity"` // decimal
		Price          string `json:"price"`    // decimal
		Excise         string `json:"excise"`   // decimal
		Measure        string `json:"measure"`
		VatType        string `json:"vatType"`
		PaymentSubject string `json:"paymentSubject"`
		PaymentMethod  string `json:"paymentMethod"`

		Product struct {
			Country     string `json:"country"`
			Declaration string `json:"declaration"`
		} `json:"product"`

		Marking struct {
			Code      string `json:"code"`
			AgentType string `json:"agentType"`

			Quantity struct {
				Numerator   int `json:"numerator"`
				Denominator int `json:"denominator"`
			}
		}

		Supplier struct {
			Name  string `json:"name"`
			INN   string `json:"INN"`
			Phone string `json:"phone"`
		}
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

func (c Checkout) ReceiptList(paymentID string) (r []Receipt, err error) {
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
