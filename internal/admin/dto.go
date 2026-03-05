package admin

import "time"

type AdminLoginRequest struct {
	Username string `json:"username" validate:"required,min=3,max=50"`
	Password string `json:"password" validate:"required,min=6"`
}

type AdminAuthResponse struct {
	Token string     `json:"token"`
	Admin *AdminData `json:"admin"`
}

type AdminData struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	FullName string `json:"full_name"`
	Role     string `json:"role"`
}

type VerificationItem struct {
	MaidID             string    `json:"maid_id"`
	FullName           string    `json:"full_name"`
	PhoneNumber        string    `json:"phone_number"`
	Bio                string    `json:"bio"`
	HourlyRate         int       `json:"hourly_rate"`
	IDNumber           string    `json:"id_number"`
	IDType             string    `json:"id_type"`
	VerificationStatus string    `json:"verification_status"`
	SubmittedAt        time.Time `json:"submitted_at"`
}

type PendingVerificationsResponse struct {
	Items []VerificationItem `json:"items"`
	Page  int                `json:"page"`
	Limit int                `json:"limit"`
	Total int                `json:"total"`
}

type VerificationDetailsResponse struct {
	MaidID             string    `json:"maid_id"`
	Bio                string    `json:"bio"`
	HourlyRate         int       `json:"hourly_rate"`
	IDNumber           string    `json:"id_number"`
	IDType             string    `json:"id_type"`
	VerificationStatus string    `json:"verification_status"`
	VideoURL           string    `json:"video_url"`
	IDPhotoURL         string    `json:"id_photo_url"`
	SubmittedAt        time.Time `json:"submitted_at"`
}

type ApproveVerificationRequest struct {
	MaidID string `json:"maid_id" validate:"required,uuid"`
}

type RejectVerificationRequest struct {
	MaidID string `json:"maid_id" validate:"required,uuid"`
	Reason string `json:"reason" validate:"required,min=10,max=500"`
}