package payment

import (
	"context"
	"log"
	"net/http"

	"kazi-backend/internal/common/util"

	"github.com/gofiber/fiber/v2"
)

// WebhookEventHandler is implemented by the booking domain so this package
// doesn't need to import it back (that would create an import cycle, since
// booking already imports payment for PaymentClient).
type WebhookEventHandler interface {
	HandlePaymentCompleted(ctx context.Context, transactionID string) error
	HandlePayoutCompleted(ctx context.Context, transactionID string) error
	HandlePaymentFailed(ctx context.Context, transactionID string) error
}

type WebhookHandler struct {
	events       WebhookEventHandler
	sharedSecret string
}

func NewWebhookHandler(events WebhookEventHandler, sharedSecret string) *WebhookHandler {
	return &WebhookHandler{
		events:       events,
		sharedSecret: sharedSecret,
	}
}

type webhookPayload struct {
	TransactionID string `json:"transaction_id"`
	EventType     string `json:"event_type"`
}

func (h *WebhookHandler) HandleWebhook(c *fiber.Ctx) error {
	if h.sharedSecret == "" || c.Get("X-Webhook-Secret") != h.sharedSecret {
		return util.ErrorResponse(c, http.StatusUnauthorized, "invalid webhook secret")
	}

	var payload webhookPayload
	if err := c.BodyParser(&payload); err != nil {
		return util.ValidationErrorResponse(c, "invalid webhook payload")
	}

	if payload.TransactionID == "" || payload.EventType == "" {
		return util.ValidationErrorResponse(c, "transaction_id and event_type are required")
	}

	ctx := c.Context()

	var err error
	switch payload.EventType {
	case "payment.completed":
		err = h.events.HandlePaymentCompleted(ctx, payload.TransactionID)
	case "payout.completed":
		err = h.events.HandlePayoutCompleted(ctx, payload.TransactionID)
	case "payment.failed":
		err = h.events.HandlePaymentFailed(ctx, payload.TransactionID)
	default:
		log.Printf("[PaymentWebhook] Ignoring unknown event_type: %s", payload.EventType)
		return util.SuccessResponse(c, nil, "ignored")
	}

	if err != nil {
		log.Printf("[PaymentWebhook] %s handling failed: %v", payload.EventType, err)
		return util.ErrorResponse(c, http.StatusInternalServerError, "failed to process webhook")
	}

	return util.SuccessResponse(c, nil, "processed")
}
