package payment

import (
	"context"
	"log/slog"
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
	HandlePaymentExpired(ctx context.Context, transactionID string) error
	HandlePaymentVoided(ctx context.Context, transactionID string) error
	HandlePayoutFailed(ctx context.Context, transactionID string) error
	HandlePayoutReversed(ctx context.Context, transactionID string) error
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
	if h.sharedSecret != "" && c.Get("X-Webhook-Secret") != h.sharedSecret {
		slog.Warn("payment webhook: rejected request with wrong shared secret",
			"remote_ip", c.IP())
		return util.ErrorResponse(c, http.StatusUnauthorized, "invalid webhook secret")
	}
	if h.sharedSecret == "" {
		slog.Warn("payment webhook: no PAYMENT_WEBHOOK_SECRET configured, accepting request unverified",
			"remote_ip", c.IP())
	}

	var payload webhookPayload
	if err := c.BodyParser(&payload); err != nil {
		slog.Warn("payment webhook: body could not be parsed",
			"error", err,
			"raw_body", string(c.Body()))
		return util.ValidationErrorResponse(c, "invalid webhook payload")
	}

	if payload.TransactionID == "" || payload.EventType == "" {
		slog.Warn("payment webhook: payload missing transaction_id or event_type",
			"event_type", payload.EventType,
			"transaction_id", payload.TransactionID)
		return util.ValidationErrorResponse(c, "transaction_id and event_type are required")
	}

	slog.Info("payment webhook: event received",
		"event_type", payload.EventType,
		"transaction_id", payload.TransactionID)

	ctx := c.Context()

	var err error
	switch payload.EventType {
	case "payment.completed":
		err = h.events.HandlePaymentCompleted(ctx, payload.TransactionID)
	case "payout.completed":
		err = h.events.HandlePayoutCompleted(ctx, payload.TransactionID)
	case "payment.failed":
		err = h.events.HandlePaymentFailed(ctx, payload.TransactionID)
	case "payment.expired":
		err = h.events.HandlePaymentExpired(ctx, payload.TransactionID)
	case "payment.voided":
		err = h.events.HandlePaymentVoided(ctx, payload.TransactionID)
	case "payout.failed":
		err = h.events.HandlePayoutFailed(ctx, payload.TransactionID)
	case "payout.reversed":
		err = h.events.HandlePayoutReversed(ctx, payload.TransactionID)
	default:
		slog.Info("payment webhook: ignoring event type we don't handle",
			"event_type", payload.EventType,
			"transaction_id", payload.TransactionID)
		return util.SuccessResponse(c, nil, "ignored")
	}

	if err != nil {
		slog.Error("payment webhook: event processing failed",
			"event_type", payload.EventType,
			"transaction_id", payload.TransactionID,
			"error", err)
		return util.ErrorResponse(c, http.StatusInternalServerError, "failed to process webhook")
	}

	slog.Info("payment webhook: event processed successfully",
		"event_type", payload.EventType,
		"transaction_id", payload.TransactionID)

	return util.SuccessResponse(c, nil, "processed")
}
