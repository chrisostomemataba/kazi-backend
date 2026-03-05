package maid

import "github.com/google/uuid"

type VerificationSubmitRequest struct {
	UserID     uuid.UUID `json:"-"`
	Bio        string    `json:"bio" validate:"max=500"`
	HourlyRate int       `json:"hourly_rate" validate:"required,min=5000,max=100000"`
	Services   []string  `json:"services" validate:"required,min=1,dive,oneof=cleaning cooking laundry childcare ironing"`
	IDNumber   string    `json:"id_number" validate:"required,min=5,max=50"`
	IDType     string    `json:"id_type" validate:"required,oneof=NIDA VOTERS_ID DRIVING_LICENSE"`
}

type MaidProfileResponse struct {
	ID                 string `json:"id"`
	UserID             string `json:"user_id"`
	Bio                string `json:"bio"`
	HourlyRate         int    `json:"hourly_rate"`
	VerificationStatus string `json:"verification_status"`
	IDNumber           string `json:"id_number"`
	IDType             string `json:"id_type"`
	RejectionReason    string `json:"rejection_reason,omitempty"`
	VideoURL           string `json:"video_url,omitempty"`
	IDPhotoURL         string `json:"id_photo_url,omitempty"`
}