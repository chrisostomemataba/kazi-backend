package maid

import (

	"github.com/google/uuid"
)

type VerificationSubmitRequest struct {
	UserID              uuid.UUID `json:"-"`
	Bio                 string    `json:"bio" validate:"max=500"`
	Gender              string    `json:"gender" validate:"required,oneof=male female other"`
	DateOfBirth         string    `json:"date_of_birth" validate:"required"` // YYYY-MM-DD format
	HomeAddress         string    `json:"home_address" validate:"required,min=10,max=500"`
	HomeLocationLat     *float64  `json:"home_location_lat" validate:"required,min=-90,max=90"`
	HomeLocationLng     *float64  `json:"home_location_lng" validate:"required,min=-180,max=180"`
	District            string    `json:"district" validate:"required,max=50"`
	Ward                string    `json:"ward" validate:"required,max=50"`
	HourlyRate          int       `json:"hourly_rate" validate:"required,min=5000,max=100000"`
	OffersContracts     bool      `json:"offers_contracts"`
	MonthlyContractRate *int      `json:"monthly_contract_rate" validate:"omitempty,min=100000,max=5000000"` // optional
	Services            []string  `json:"services" validate:"required,min=1,dive,oneof=cleaning cooking laundry childcare ironing"`
	IDNumber            string    `json:"id_number" validate:"required,min=5,max=50"`
	IDType              string    `json:"id_type" validate:"required,oneof=NIDA VOTERS_ID DRIVING_LICENSE"`
}

type UpdateLocationRequest struct {
	HomeAddress     string   `json:"home_address" validate:"required,min=10,max=500"`
	HomeLocationLat *float64 `json:"home_location_lat" validate:"required,min=-90,max=90"`
	HomeLocationLng *float64 `json:"home_location_lng" validate:"required,min=-180,max=180"`
	District        string   `json:"district" validate:"required,max=50"`
	Ward            string   `json:"ward" validate:"required,max=50"`
}

type UpdateContractRateRequest struct {
	OffersContracts     bool `json:"offers_contracts"`
	MonthlyContractRate *int `json:"monthly_contract_rate" validate:"omitempty,min=100000,max=5000000"`
}

type MaidProfileResponse struct {
	ID                  string              `json:"id"`
	UserID              string              `json:"user_id"`
	FullName            string              `json:"full_name"`
	ProfilePhotoURL     string              `json:"profile_photo_url,omitempty"`
	IsAvailableNow      bool                `json:"is_available_now"`
	Bio                 string              `json:"bio"`
	Gender              string              `json:"gender"`
	DateOfBirth         string              `json:"date_of_birth,omitempty"`
	HomeAddress         string              `json:"home_address"`
	HomeLocationLat     *float64            `json:"home_location_lat,omitempty"`
	HomeLocationLng     *float64            `json:"home_location_lng,omitempty"`
	District            string              `json:"district"`
	Ward                string              `json:"ward"`
	HourlyRate          int                 `json:"hourly_rate"`
	OffersContracts     bool                `json:"offers_contracts"`
	MonthlyContractRate *int                `json:"monthly_contract_rate,omitempty"`
	Services            []string            `json:"services"`
	VerificationStatus  string              `json:"verification_status"`
	IDNumber            string              `json:"id_number,omitempty"`
	IDType              string              `json:"id_type,omitempty"`
	RejectionReason     string              `json:"rejection_reason,omitempty"`
	VideoURL            string              `json:"video_url,omitempty"`
	IDPhotoURL          string              `json:"id_photo_url,omitempty"`
	Statistics          *MaidStatsResponse  `json:"statistics,omitempty"`
}

type MaidStatsResponse struct {
	AverageRating           float64 `json:"average_rating"`
	TotalReviews            int     `json:"total_reviews"`
	TotalJobsCompleted      int     `json:"total_jobs_completed"`
	TotalContractsCompleted int     `json:"total_contracts_completed"`
}

type SearchMaidsRequest struct {
	Latitude        *float64 `json:"latitude" validate:"omitempty,min=-90,max=90"`
	Longitude       *float64 `json:"longitude" validate:"omitempty,min=-180,max=180"`
	RadiusKM        float64  `json:"radius_km" validate:"omitempty,min=1,max=50"`
	ServiceType     string   `json:"service_type" validate:"omitempty,oneof=cleaning cooking laundry childcare ironing"`
	MinHourlyRate   *int     `json:"min_hourly_rate" validate:"omitempty,min=0"`
	MaxHourlyRate   *int     `json:"max_hourly_rate" validate:"omitempty,min=0"`
	OffersContracts *bool    `json:"offers_contracts"`
	Gender          string   `json:"gender" validate:"omitempty,oneof=male female"`
	MinRating       *float64 `json:"min_rating" validate:"omitempty,min=0,max=5"`
	Page            int      `json:"page" validate:"omitempty,min=1"`
	Limit           int      `json:"limit" validate:"omitempty,min=1,max=50"`
}

type MaidSearchResult struct {
	ID                  string   `json:"id"`
	FullName            string   `json:"full_name"`
	ProfilePhotoURL     string   `json:"profile_photo_url,omitempty"`
	Gender              string   `json:"gender"`
	District            string   `json:"district"`
	Ward                string   `json:"ward"`
	DistanceKM          *float64 `json:"distance_km,omitempty"`
	HourlyRate          int      `json:"hourly_rate"`
	OffersContracts     bool     `json:"offers_contracts"`
	MonthlyContractRate *int     `json:"monthly_contract_rate,omitempty"`
	Services            []string `json:"services"`
	AverageRating       float64  `json:"average_rating"`
	TotalReviews        int      `json:"total_reviews"`
	TotalJobsCompleted  int      `json:"total_jobs_completed"`
}

type WalletResponse struct {
	AvailableBalance int `json:"available_balance"`
	PendingAmount    int `json:"pending_amount"`
	TotalEarned      int `json:"total_earned"`
	TotalWithdrawn   int `json:"total_withdrawn"`
}

type WalletTransactionResponse struct {
	ID               string  `json:"id"`
	TransactionType  string  `json:"transaction_type"`
	Amount           int     `json:"amount"`
	RelatedBookingID *string `json:"related_booking_id,omitempty"`
	CreatedAt        string  `json:"created_at"`
}