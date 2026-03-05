package sms

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type SMSService struct {
	APIToken string
	SenderID string
	BaseURL  string
	client   *http.Client
}

type NotifyAfricaRequest struct {
	PhoneNumber string `json:"phone_number"`
	Message     string `json:"message"`
	SenderID    string `json:"sender_id"`
}

type NotifyAfricaResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func NewSMSService(apiToken, senderID, baseURL string) *SMSService {
	return &SMSService{
		APIToken: apiToken,
		SenderID: senderID,
		BaseURL:  baseURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (s *SMSService) SendOTP(phoneNumber, code string) error {
	message := fmt.Sprintf("KAZI: Your verification code is %s. Valid for 10 minutes.", code)
	return s.send(phoneNumber, message)
}

func (s *SMSService) SendPasswordResetOTP(phoneNumber, code string) error {
	message := fmt.Sprintf("KAZI: Your password reset code is %s. Valid for 10 minutes.", code)
	return s.send(phoneNumber, message)
}

func (s *SMSService) send(phoneNumber, message string) error {
	payload := NotifyAfricaRequest{
		PhoneNumber: phoneNumber,
		Message:     message,
		SenderID:    s.SenderID,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", s.BaseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.APIToken))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	log.Printf("[SMS] Sending to: %s", phoneNumber)
	log.Printf("[SMS] Payload: %s", string(jsonData))

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	log.Printf("[SMS] Response Status: %d", resp.StatusCode)
	log.Printf("[SMS] Response Body: %s", string(body))

	// Notify Africa returns 200 or 202 on success
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("SMS API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Try to parse response
	var apiResponse NotifyAfricaResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		// If can't parse JSON but status is 200/202, consider it success
		log.Printf("[SMS] Warning: Could not parse response JSON, but status code indicates success")
		return nil
	}

	// Check if API returned success in response body
	if !apiResponse.Success {
		return fmt.Errorf("SMS API error: %s", apiResponse.Message)
	}

	log.Printf("[SMS] Successfully sent to: %s", phoneNumber)
	return nil
}