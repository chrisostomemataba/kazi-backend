package customer

type AddLocationRequest struct {
	Label     string   `json:"label" validate:"required,oneof=Home Work Other"`
	Address   string   `json:"address" validate:"required,min=10,max=500"`
	Latitude  float64  `json:"latitude" validate:"required,min=-90,max=90"`
	Longitude float64  `json:"longitude" validate:"required,min=-180,max=180"`
	District  string   `json:"district" validate:"max=50"`
	Ward      string   `json:"ward" validate:"max=50"`
	IsDefault bool     `json:"is_default"`
}

type CustomerLocationResponse struct {
	ID        string  `json:"id"`
	Label     string  `json:"label"`
	Address   string  `json:"address"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	District  string  `json:"district"`
	Ward      string  `json:"ward"`
	IsDefault bool    `json:"is_default"`
}

type CustomerProfileResponse struct {
	ID                string                      `json:"id"`
	UserID            string                      `json:"user_id"`
	FullName          string                      `json:"full_name"`
	PhoneNumber       string                      `json:"phone_number"`
	PreferredLanguage string                      `json:"preferred_language"`
	SavedLocations    []CustomerLocationResponse  `json:"saved_locations"`
	Statistics        *CustomerStatsResponse      `json:"statistics"`
}

type CustomerStatsResponse struct {
	AverageMaidRating float64 `json:"average_maid_rating"`
	TotalBookings     int     `json:"total_bookings"`
	CompletedBookings int     `json:"completed_bookings"`
	PaymentOnTimeRate float64 `json:"payment_on_time_rate"`
}