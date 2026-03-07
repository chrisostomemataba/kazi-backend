package auth

type RequestOTPRequest struct {
	PhoneNumber string `json:"phone_number" validate:"required,len=12"`
}

type VerifyOTPRequest struct {
	PhoneNumber string `json:"phone_number" validate:"required,len=12"`
	Code        string `json:"code" validate:"required,len=6"`
}

type CompleteProfileRequest struct {
	FullName string   `json:"full_name" validate:"required,min=2,max=100"`
	Roles    []string `json:"roles" validate:"required,min=1,dive,oneof=customer maid"`
}

type AuthResponse struct {
	Token string    `json:"token"`
	User  *UserData `json:"user"`
}

type LoginRequest struct {
	PhoneNumber string `json:"phone_number" validate:"required,len=12"`
	OTPCode     string `json:"otp_code" validate:"required,len=6"`
}


type UserData struct {
	ID              string   `json:"id"`
	PhoneNumber     string   `json:"phone_number"`
	FullName        string   `json:"full_name"`
	ProfilePhotoURL string   `json:"profile_photo_url,omitempty"`
	Roles           []string `json:"roles"`
}