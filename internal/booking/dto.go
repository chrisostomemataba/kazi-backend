package booking

type CreateBookingRequest struct {
	MaidID              string              `json:"maid_id" validate:"required,uuid"`
	ServiceType         string              `json:"service_type" validate:"required,oneof=cleaning cooking laundry childcare ironing"`
	BookingDate         string              `json:"booking_date" validate:"required"`
	StartTime           string              `json:"start_time" validate:"required"`
	EndTime             string              `json:"end_time" validate:"required"`
	ServiceLocation     ServiceLocationData `json:"service_location" validate:"required"`
	SpecialInstructions string              `json:"special_instructions" validate:"max=1000"`
}

type ServiceLocationData struct {
	Address   string  `json:"address" validate:"required,min=5,max=500"`
	Latitude  float64 `json:"latitude" validate:"required,min=-90,max=90"`
	Longitude float64 `json:"longitude" validate:"required,min=-180,max=180"`
	District  string  `json:"district" validate:"max=50"`
	Ward      string  `json:"ward" validate:"max=50"`
}

type ValidateBookingRequest struct {
	MaidID      string `json:"maid_id" validate:"required,uuid"`
	BookingDate string `json:"booking_date" validate:"required"`
	StartTime   string `json:"start_time" validate:"required"`
	EndTime     string `json:"end_time" validate:"required"`
}

type ValidateBookingResponse struct {
	CanBook       bool     `json:"can_book"`
	MaidAvailable bool     `json:"maid_available"`
	CustomerReady bool     `json:"customer_ready"`
	Issues        []string `json:"issues,omitempty"`
}

// Workflow C3
type DeclineBookingRequest struct {
	Reason string `json:"reason" validate:"max=500"`
}

// Workflow D
type InitiatePaymentRequest struct {
	Provider    string `json:"provider" validate:"required,oneof=Mpesa TigoPesa AirtelMoney Halopesa"`
	PhoneNumber string `json:"phone_number" validate:"required,len=12"`
}

// Workflow E1
type ArrivalRequest struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// Response types

type BookingResponse struct {
	ID              string                `json:"id"`
	ReferenceNumber string                `json:"reference_number"`
	Status          string                `json:"booking_status"`
	PaymentStatus   string                `json:"payment_status"`
	ServiceType     string                `json:"service_type"`
	BookingDate     string                `json:"booking_date"`
	StartTime       string                `json:"start_time"`
	EndTime         string                `json:"end_time"`
	DurationHours   float64               `json:"duration_hours"`
	ServiceStartedAt *string               `json:"service_started_at,omitempty"`
	ServiceCompletedAt *string               `json:"service_completed_at,omitempty"`
	Maid            *BookingMaidData      `json:"maid,omitempty"`
	Customer        *BookingCustomerData  `json:"customer,omitempty"`
	Location        *BookingLocationData  `json:"location,omitempty"`
	Pricing         *BookingPricingData   `json:"pricing,omitempty"`
	Timeline        []BookingTimelineItem `json:"timeline,omitempty"`
}

type BookingMaidData struct {
	ID              string  `json:"id"`
	FullName        string  `json:"full_name"`
	PhoneNumber     string  `json:"phone_number"`
	ProfilePhotoURL string  `json:"profile_photo_url,omitempty"`
	AverageRating   float64 `json:"average_rating"`
}

type BookingCustomerData struct {
	ID                string  `json:"id"`
	FullName          string  `json:"full_name"`
	PhoneNumber       string  `json:"phone_number"`
	AverageMaidRating float64 `json:"average_maid_rating"`
}

type BookingLocationData struct {
	Address           string   `json:"address"`
	Latitude          float64  `json:"latitude"`
	Longitude         float64  `json:"longitude"`
	District          string   `json:"district"`
	Ward              string   `json:"ward"`
	ArrivalVerified   bool     `json:"arrival_verified"`
	ArrivalVerifiedAt *string  `json:"arrival_verified_at,omitempty"`
}

type BookingPricingData struct {
	HourlyRate      int     `json:"hourly_rate"`
	Subtotal        int     `json:"subtotal"`
	PlatformFee     int     `json:"platform_fee"`
	PlatformFeeRate float64 `json:"platform_fee_rate"`
	Total           int     `json:"total"`
	MaidPayout      int     `json:"maid_payout"`
}

type BookingTimelineItem struct {
	EventType      string  `json:"event_type"`
	EventTimestamp string  `json:"event_timestamp"`
	TriggeredBy    *string `json:"triggered_by,omitempty"`
	Notes          string  `json:"notes,omitempty"`
}

type PaymentResponse struct {
	PaymentInitiated bool   `json:"payment_initiated"`
	TransactionID    string `json:"transaction_id"`
	Message          string `json:"message"`
}