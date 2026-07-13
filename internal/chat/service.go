package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"kazi-backend/internal/auth"
	"kazi-backend/internal/booking"
	wsHub "kazi-backend/internal/common/websocket"
	"kazi-backend/internal/notification"

	"github.com/google/uuid"
)

// Chat opens once the maid accepts and stays open through completion,
// so both sides can still coordinate reviews or report problems.
var chatEnabledStatuses = map[string]bool{
	"maid_accepted":   true,
	"payment_pending": true,
	"confirmed":       true,
	"in_progress":     true,
	"completed":       true,
}

type wsFrame struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type Service struct {
	repo            *Repository
	bookingRepo     *booking.Repository
	authRepo        *auth.Repository
	hub             *wsHub.Hub
	notificationSvc *notification.Service
}

func NewService(
	repo *Repository,
	bookingRepo *booking.Repository,
	authRepo *auth.Repository,
	hub *wsHub.Hub,
	notificationSvc *notification.Service,
) *Service {
	return &Service{
		repo:            repo,
		bookingRepo:     bookingRepo,
		authRepo:        authRepo,
		hub:             hub,
		notificationSvc: notificationSvc,
	}
}

func (s *Service) SendMessage(ctx context.Context, senderID uuid.UUID, bookingID string, req *SendMessageRequest) (*MessageResponse, error) {
	chatBooking, otherUserID, accessError := s.resolveConversation(ctx, senderID, bookingID)
	if accessError != nil {
		return nil, accessError
	}

	canChat := chatEnabledStatuses[chatBooking.BookingStatus]
	if !canChat {
		return nil, fmt.Errorf("chat is not available for booking in status: %s", chatBooking.BookingStatus)
	}

	newMessage := &ChatMessage{
		BookingID:   chatBooking.ID,
		SenderID:    senderID,
		RecipientID: otherUserID,
		Body:        req.Body,
	}

	createError := s.repo.Create(ctx, newMessage)
	if createError != nil {
		return nil, fmt.Errorf("failed to save message: %w", createError)
	}

	messageResponse := buildMessageResponse(newMessage)

	s.pushFrame(otherUserID, "chat.message", messageResponse)

	recipientIsOffline := !s.hub.IsConnected(otherUserID)
	if recipientIsOffline {
		senderName := s.lookupUserName(ctx, senderID)
		s.notificationSvc.NotifyNewChatMessage(ctx, otherUserID, senderName)
	}

	slog.Info("chat: message sent",
		"booking_reference", chatBooking.ReferenceNumber,
		"sender_id", senderID.String(),
		"recipient_online", !recipientIsOffline)

	return messageResponse, nil
}

// GetMessages returns the newest page of the conversation (ascending within
// the page) and marks everything addressed to the reader as read — opening
// the conversation is what "reading" means.
func (s *Service) GetMessages(ctx context.Context, readerID uuid.UUID, bookingID string, limit, offset int) ([]MessageResponse, error) {
	chatBooking, _, accessError := s.resolveConversation(ctx, readerID, bookingID)
	if accessError != nil {
		return nil, accessError
	}

	messages, listError := s.repo.ListByBooking(ctx, chatBooking.ID, limit, offset)
	if listError != nil {
		return nil, fmt.Errorf("failed to fetch messages: %w", listError)
	}

	markReadError := s.markConversationRead(ctx, chatBooking, readerID)
	if markReadError != nil {
		slog.Warn("chat: failed to mark conversation read", "error", markReadError)
	}

	responses := make([]MessageResponse, 0, len(messages))
	for i := len(messages) - 1; i >= 0; i-- {
		responses = append(responses, *buildMessageResponse(&messages[i]))
	}

	return responses, nil
}

func (s *Service) MarkRead(ctx context.Context, readerID uuid.UUID, bookingID string) error {
	chatBooking, _, accessError := s.resolveConversation(ctx, readerID, bookingID)
	if accessError != nil {
		return accessError
	}

	return s.markConversationRead(ctx, chatBooking, readerID)
}

// GetConversations lists every chat-eligible booking the user participates in
// (as customer, maid, or both), decorated with the other party, the last
// message, and the unread count — newest activity first.
func (s *Service) GetConversations(ctx context.Context, userID uuid.UUID) ([]ConversationResponse, error) {
	customerBookings, customerError := s.bookingRepo.GetCustomerBookings(ctx, userID, "", 1, 50)
	if customerError != nil {
		return nil, fmt.Errorf("failed to fetch customer bookings: %w", customerError)
	}

	maidBookings, maidError := s.bookingRepo.GetMaidBookings(ctx, userID, "", "", 1, 50)
	if maidError != nil {
		return nil, fmt.Errorf("failed to fetch maid bookings: %w", maidError)
	}

	allBookings := append(customerBookings, maidBookings...)

	conversations := make([]ConversationResponse, 0, len(allBookings))
	for i := range allBookings {
		conversationBooking := &allBookings[i]

		isChatEnabled := chatEnabledStatuses[conversationBooking.BookingStatus]
		if !isChatEnabled {
			continue
		}

		otherUserID := conversationBooking.MaidID
		if conversationBooking.MaidID == userID {
			otherUserID = conversationBooking.CustomerID
		}

		otherUser, findUserError := s.authRepo.FindUserByID(ctx, otherUserID)
		if findUserError != nil {
			continue
		}

		conversation := ConversationResponse{
			BookingID:     conversationBooking.ID.String(),
			BookingRef:    conversationBooking.ReferenceNumber,
			ServiceType:   conversationBooking.ServiceType,
			BookingStatus: conversationBooking.BookingStatus,
			BookingDate:   conversationBooking.BookingDate.Format("2006-01-02"),
			OtherUser: ConversationUser{
				ID:       otherUser.ID.String(),
				FullName: otherUser.FullName,
				PhotoURL: otherUser.ProfilePhotoURL,
			},
		}

		lastMessage, lastMessageError := s.repo.GetLastMessage(ctx, conversationBooking.ID)
		if lastMessageError == nil && lastMessage != nil {
			lastMessageAt := lastMessage.CreatedAt.Format(time.RFC3339)
			conversation.LastMessage = lastMessage.Body
			conversation.LastMessageAt = &lastMessageAt
		}

		unreadCount, unreadError := s.repo.CountUnreadByBooking(ctx, conversationBooking.ID, userID)
		if unreadError == nil {
			conversation.UnreadCount = unreadCount
		}

		conversations = append(conversations, conversation)
	}

	sort.SliceStable(conversations, func(a, b int) bool {
		aHasMessage := conversations[a].LastMessageAt != nil
		bHasMessage := conversations[b].LastMessageAt != nil
		if aHasMessage && bHasMessage {
			return *conversations[a].LastMessageAt > *conversations[b].LastMessageAt
		}
		return aHasMessage && !bHasMessage
	})

	return conversations, nil
}

func (s *Service) GetUnreadCount(ctx context.Context, userID uuid.UUID) (*UnreadCountResponse, error) {
	unreadCount, countError := s.repo.CountUnreadForUser(ctx, userID)
	if countError != nil {
		return nil, fmt.Errorf("failed to count unread messages: %w", countError)
	}
	return &UnreadCountResponse{UnreadCount: unreadCount}, nil
}

// ── Internals ─────────────────────────────────────────────────────────────────

func (s *Service) resolveConversation(ctx context.Context, userID uuid.UUID, bookingID string) (*booking.Booking, uuid.UUID, error) {
	bookingUUID, parseError := uuid.Parse(bookingID)
	if parseError != nil {
		return nil, uuid.Nil, errors.New("invalid booking ID")
	}

	chatBooking, findError := s.bookingRepo.GetBookingByID(ctx, bookingUUID)
	if findError != nil {
		return nil, uuid.Nil, errors.New("booking not found")
	}

	isCustomer := chatBooking.CustomerID == userID
	isMaid := chatBooking.MaidID == userID
	if !isCustomer && !isMaid {
		return nil, uuid.Nil, errors.New("only booking participants can use this chat")
	}

	otherUserID := chatBooking.MaidID
	if isMaid {
		otherUserID = chatBooking.CustomerID
	}

	return chatBooking, otherUserID, nil
}

func (s *Service) markConversationRead(ctx context.Context, chatBooking *booking.Booking, readerID uuid.UUID) error {
	readCount, markError := s.repo.MarkAllRead(ctx, chatBooking.ID, readerID, time.Now())
	if markError != nil {
		return fmt.Errorf("failed to mark messages read: %w", markError)
	}

	hasNewReads := readCount > 0
	if hasNewReads {
		otherUserID := chatBooking.MaidID
		if chatBooking.MaidID == readerID {
			otherUserID = chatBooking.CustomerID
		}
		s.pushFrame(otherUserID, "chat.read", map[string]string{
			"booking_id": chatBooking.ID.String(),
			"reader_id":  readerID.String(),
		})
	}

	return nil
}

func (s *Service) pushFrame(userID uuid.UUID, frameType string, data interface{}) {
	framePayload, marshalError := json.Marshal(wsFrame{Type: frameType, Data: data})
	if marshalError != nil {
		slog.Warn("chat: failed to marshal websocket frame", "error", marshalError)
		return
	}
	s.hub.SendToUser(userID, framePayload)
}

func (s *Service) lookupUserName(ctx context.Context, userID uuid.UUID) string {
	user, findError := s.authRepo.FindUserByID(ctx, userID)
	if findError != nil || user == nil {
		return "Mtumiaji"
	}
	return user.FullName
}

func buildMessageResponse(message *ChatMessage) *MessageResponse {
	response := &MessageResponse{
		ID:        message.ID.String(),
		BookingID: message.BookingID.String(),
		SenderID:  message.SenderID.String(),
		Body:      message.Body,
		CreatedAt: message.CreatedAt.Format(time.RFC3339),
	}
	if message.ReadAt != nil {
		readAt := message.ReadAt.Format(time.RFC3339)
		response.ReadAt = &readAt
	}
	return response
}
