package paymaster

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"time"
)

type (
	Client struct {
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

	ReceiptRequest struct {
		PaymentID string       `json:"paymentId"`
		Amount    Amount       `json:"amount"`
		Type      string       `json:"type"`
		Client    Client       `json:"client"`
		Items     ReceiptItems `json:"items"`
	}
)
type Receipt struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created"`
	PaymentID string    `json:"paymentId"`
	Amount    Amount    `json:"amount"`
	Type      string    `json:"type"`
	Status    string    `json:"status"`
}

func (c Checkout) CreateReceipt(receipt ReceiptRequest) (r *Receipt, _ error) {
	data, err := json.Marshal(receipt)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, c.BaseURL+"/receipts", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", c.AuthToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	return r, json.NewDecoder(resp.Body).Decode(r)
}

func (c Checkout) ReceiptByID(id string) (r *Receipt, err error) {
	req, err := http.NewRequest(http.MethodPost, c.BaseURL+"/receipts/"+id, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", c.AuthToken)

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
	url := c.BaseURL + "/receipts" + params.Encode()

	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", c.AuthToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	return r, json.NewDecoder(resp.Body).Decode(&r)
}
