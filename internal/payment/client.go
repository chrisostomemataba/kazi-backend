package payment

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

type PaymentClient struct {
	baseUrl    string
	httpClient *http.Client
}

func NewPaymentClient(baseUrl string) *PaymentClient {
	return &PaymentClient{
		baseUrl:    baseUrl,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *PaymentClient) doRequest(method, path string, body interface{}) (*http.Response, error) {
	token, err := GetPaymentServiceToken()
	if err != nil {
		return nil, fmt.Errorf("get payment service token: %w", err)
	}

	var reqBody io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(payload)
	}

	req, err := http.NewRequest(method, c.baseUrl+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("build payment request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	return c.httpClient.Do(req)
}

// decodeJSONResponse reads a payment-service response, treating non-2xx status
// codes as errors and unmarshaling the body into out otherwise.
func decodeJSONResponse(resp *http.Response, out interface{}) error {
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read payment response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("payment service returned %d: %s", resp.StatusCode, string(body))
	}

	if out == nil {
		return nil
	}

	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("parse payment response: %w", err)
	}

	return nil
}

type collectFromCustomerRequest struct {
	PhoneNumber       string                 `json:"phone_number"`
	Amount            int                    `json:"amount"`
	CustomerFirstname string                 `json:"customer_firstname"`
	CustomerLastname  string                 `json:"customer_lastname"`
	CustomerEmail     string                 `json:"customer_email"`
	Metadata          map[string]interface{} `json:"metadata"`
}

type collectFromCustomerResponse struct {
	TransactionID string `json:"transaction_id"`
}

// CollectFromCustomer initiates a mobile money collection from the customer
// after the maid has accepted the booking.
func (c *PaymentClient) CollectFromCustomer(
	phoneNumber string,
	amount int,
	customerFirstname string,
	customerLastname string,
	customerEmail string,
	metadata map[string]interface{},
) (string, error) {
	slog.Info("payment service: initiating mobile money collection",
		"amount_tzs", amount,
		"customer_name", customerFirstname+" "+customerLastname)

	resp, err := c.doRequest(http.MethodPost, "/v1/collect/mobile", collectFromCustomerRequest{
		PhoneNumber:       phoneNumber,
		Amount:            amount,
		CustomerFirstname: customerFirstname,
		CustomerLastname:  customerLastname,
		CustomerEmail:     customerEmail,
		Metadata:          metadata,
	})
	if err != nil {
		slog.Error("payment service: mobile collection request failed", "error", err)
		return "", err
	}

	var out collectFromCustomerResponse
	if err := decodeJSONResponse(resp, &out); err != nil {
		slog.Error("payment service: mobile collection response invalid", "error", err)
		return "", err
	}

	slog.Info("payment service: mobile collection accepted, waiting for customer PIN",
		"transaction_id", out.TransactionID)

	return out.TransactionID, nil
}

type collectFromCustomerByCardRequest struct {
	PhoneNumber     string                 `json:"phone_number"`
	Amount          int                    `json:"amount"`
	BillingAddress  string                 `json:"billing_address"`
	BillingCity     string                 `json:"billing_city"`
	BillingState    string                 `json:"billing_state"`
	BillingPostcode string                 `json:"billing_postcode"`
	BillingCountry  string                 `json:"billing_country"`
	RedirectURL     string                 `json:"redirect_url"`
	CancelURL       string                 `json:"cancel_url"`
	Metadata        map[string]interface{} `json:"metadata"`
}

type collectFromCustomerByCardResponse struct {
	TransactionID string `json:"transaction_id"`
	PaymentURL    string `json:"payment_url"`
}

// CollectFromCustomerByCard starts a hosted card checkout. Snippe requires
// billing details for card payments, and the customer completes the payment
// on the returned payment_url — the mobile app must open that URL in a
// WebView or external browser, then rely on the payment.completed webhook.
func (c *PaymentClient) CollectFromCustomerByCard(
	phoneNumber string,
	amount int,
	billingAddress string,
	billingCity string,
	billingState string,
	billingPostcode string,
	billingCountry string,
	redirectURL string,
	cancelURL string,
	metadata map[string]interface{},
) (string, string, error) {
	slog.Info("payment service: initiating card collection",
		"amount_tzs", amount,
		"billing_city", billingCity,
		"billing_country", billingCountry)

	resp, err := c.doRequest(http.MethodPost, "/v1/collect/card", collectFromCustomerByCardRequest{
		PhoneNumber:     phoneNumber,
		Amount:          amount,
		BillingAddress:  billingAddress,
		BillingCity:     billingCity,
		BillingState:    billingState,
		BillingPostcode: billingPostcode,
		BillingCountry:  billingCountry,
		RedirectURL:     redirectURL,
		CancelURL:       cancelURL,
		Metadata:        metadata,
	})
	if err != nil {
		slog.Error("payment service: card collection request failed", "error", err)
		return "", "", err
	}

	var out collectFromCustomerByCardResponse
	if err := decodeJSONResponse(resp, &out); err != nil {
		slog.Error("payment service: card collection response invalid", "error", err)
		return "", "", err
	}

	if out.PaymentURL == "" {
		slog.Error("payment service: card collection returned no payment_url",
			"transaction_id", out.TransactionID)
		return "", "", fmt.Errorf("card collection returned no payment_url")
	}

	slog.Info("payment service: card checkout created, customer must open payment page",
		"transaction_id", out.TransactionID)

	return out.TransactionID, out.PaymentURL, nil
}

type holdEscrowRequest struct {
	CollectionTransactionID string                 `json:"collection_transaction_id"`
	BookingReference        string                 `json:"booking_reference"`
	HoldAmount              int                    `json:"hold_amount"`
	Metadata                map[string]interface{} `json:"metadata"`
}

type holdEscrowResponse struct {
	EscrowHoldID string `json:"escrow_hold_id"`
}

// HoldEscrow moves a completed collection into escrow after payment.completed
// is confirmed via webhook.
func (c *PaymentClient) HoldEscrow(
	collectionTransactionID string,
	bookingReference string,
	holdAmount int,
	metadata map[string]interface{},
) (string, error) {
	slog.Info("payment service: holding funds in escrow",
		"collection_transaction_id", collectionTransactionID,
		"booking_reference", bookingReference,
		"hold_amount_tzs", holdAmount)

	resp, err := c.doRequest(http.MethodPost, "/v1/escrow/hold", holdEscrowRequest{
		CollectionTransactionID: collectionTransactionID,
		BookingReference:        bookingReference,
		HoldAmount:              holdAmount,
		Metadata:                metadata,
	})
	if err != nil {
		slog.Error("payment service: escrow hold request failed",
			"booking_reference", bookingReference,
			"error", err)
		return "", err
	}

	var out holdEscrowResponse
	if err := decodeJSONResponse(resp, &out); err != nil {
		slog.Error("payment service: escrow hold response invalid",
			"booking_reference", bookingReference,
			"error", err)
		return "", err
	}

	slog.Info("payment service: funds held in escrow",
		"booking_reference", bookingReference,
		"escrow_hold_id", out.EscrowHoldID)

	return out.EscrowHoldID, nil
}

type releaseEscrowRequest struct {
	CollectionTransactionID string `json:"collection_transaction_id"`
	RecipientPhone          string `json:"recipient_phone"`
	RecipientName           string `json:"recipient_name"`
	Narration               string `json:"narration"`
}

type releaseEscrowResponse struct {
	DisbursementTransactionID string `json:"disbursement_transaction_id"`
}

// ReleaseEscrow disburses held escrow funds to the maid once the customer
// confirms job completion.
func (c *PaymentClient) ReleaseEscrow(
	collectionTransactionID string,
	recipientPhone string,
	recipientName string,
	narration string,
) (string, error) {
	slog.Info("payment service: releasing escrow to maid",
		"collection_transaction_id", collectionTransactionID,
		"recipient_name", recipientName,
		"narration", narration)

	resp, err := c.doRequest(http.MethodPost, "/v1/escrow/release", releaseEscrowRequest{
		CollectionTransactionID: collectionTransactionID,
		RecipientPhone:          recipientPhone,
		RecipientName:           recipientName,
		Narration:               narration,
	})
	if err != nil {
		slog.Error("payment service: escrow release request failed",
			"collection_transaction_id", collectionTransactionID,
			"error", err)
		return "", err
	}

	var out releaseEscrowResponse
	if err := decodeJSONResponse(resp, &out); err != nil {
		slog.Error("payment service: escrow release response invalid",
			"collection_transaction_id", collectionTransactionID,
			"error", err)
		return "", err
	}

	slog.Info("payment service: escrow release accepted, disbursement in flight",
		"disbursement_transaction_id", out.DisbursementTransactionID)

	return out.DisbursementTransactionID, nil
}

type transactionStatusResponse struct {
	Status string `json:"status"`
}

// GetTransactionStatus checks whether a collection transaction has completed.
func (c *PaymentClient) GetTransactionStatus(transactionID string) (string, error) {
	resp, err := c.doRequest(http.MethodGet, "/v1/transactions/"+transactionID, nil)
	if err != nil {
		return "", err
	}

	var out transactionStatusResponse
	if err := decodeJSONResponse(resp, &out); err != nil {
		return "", err
	}

	return out.Status, nil
}
