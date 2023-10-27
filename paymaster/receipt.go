package paymaster

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	Email string `json:"email"`
	Phone string `json:"phone"`
	Name  string `json:"name"`
	INN   string `json:"INN"`
}

type ReceiptRequest struct {
	PaymentID string `json:"paymentId"`
	Amount    Amount `json:"amount"`
	Type      string `json:"type"`
	Client    Client `json:"client"`
	Items     struct {
		Name string `json:"name"`

		// decimals
		Quantity string `json:"quantity"`
		Price    string `json:"price"`
		Excise   string `json:"excise"`

		Measure string `json:"measure"`
		Product struct {
			Country     string `json:"country"`
			Declaration string `json:"declaration"`
		} `json:"product"`
		VatType        string `json:"vatType"`
		PaymentSubject string `json:"paymentSubject"`
		PaymentMethod  string `json:"paymentMethod"`
		Marking        struct {
			Code     string `json:"code"`
			Quantity struct {
				Numerator   int `json:"numerator"`
				Denominator int `json:"denominator"`
			}
			AgentType string `json:"agentType"`
		}
		Supplier struct {
			Name  string `json:"name"`
			INN   string `json:"INN"`
			Phone string `json:"phone"`
		}
	} `json:"items"`
}

type Receipt struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created"`
	PaymentID string    `json:"paymentId"`
	Amount    Amount    `json:"amount"`
	Type      string    `json:"type"`
	Status    string    `json:"status"`
}

func (c Checkout) CreateReceipt(receipt ReceiptRequest) (r *Receipt, err error) {
	endpoint := c.BaseURL + "/receipts"

	data, err := json.Marshal(receipt)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", c.AuthorizationBearer)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(r)
	return
}

func (c Checkout) ReceiptByID(id string) (r *Receipt, err error) {
	endpoint := c.BaseURL + "/receipts/" + id

	req, err := http.NewRequest(http.MethodPost, endpoint, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", c.AuthorizationBearer)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(r)
	return
}

func (c Checkout) ReceiptList(paymentID string) (r []Receipt, err error) {
	endpoint := c.BaseURL + "/receipts"

	params := url.Values{}
	params.Set("paymentId", paymentID)

	req, err := http.NewRequest(http.MethodPost, endpoint+params.Encode(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", c.AuthorizationBearer)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&r)
	return
}
