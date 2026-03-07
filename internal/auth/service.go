package auth

import (
	"errors"
	"fmt"
	"math/rand"
	"time"

	"kazi-backend/internal/common/sms"
	"kazi-backend/internal/common/util"

	"gorm.io/gorm"
)

type Service struct {
	repo       *Repository
	smsService *sms.SMSService
	jwtSecret  string
}

func NewService(repo *Repository, smsService *sms.SMSService, jwtSecret string) *Service {
	return &Service{
		repo:       repo,
		smsService: smsService,
		jwtSecret:  jwtSecret,
	}
}

func (s *Service) RequestOTP(phoneNumber string) error {
	formattedPhone := util.FormatPhoneNumber(phoneNumber)

	// Rate limiting: max 3 OTPs per 10 minutes
	recentCount, err := s.repo.CountRecentOTPs(formattedPhone, time.Now().Add(-10*time.Minute))
	if err != nil {
		return err
	}
	if recentCount >= 3 {
		return errors.New("too many OTP requests, please try again in 10 minutes")
	}

	code := generateOTPCode()
	otp := &OTPCode{
		PhoneNumber: formattedPhone,
		Code:        code,
		Purpose:     "login",
		ExpiresAt:   time.Now().Add(10 * time.Minute),
	}

	if err := s.repo.CreateOTP(otp); err != nil {
		return fmt.Errorf("failed to create OTP: %w", err)
	}

	// Send SMS via Notify Africa
	if err := s.smsService.SendOTP(formattedPhone, code); err != nil {
		return fmt.Errorf("failed to send SMS: %w", err)
	}

	return nil
}

func (s *Service) VerifyOTP(phoneNumber, code string) (bool, error) {
	formattedPhone := util.FormatPhoneNumber(phoneNumber)

	otp, err := s.repo.FindValidOTP(formattedPhone, code)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, errors.New("invalid or expired OTP")
		}
		return false, err
	}

	if otp.Attempts >= 3 {
		return false, errors.New("OTP attempt limit exceeded")
	}

	// Mark OTP as used
	if err := s.repo.MarkOTPAsUsed(otp.ID); err != nil {
		return false, err
	}

	// Check if user exists
	user, err := s.repo.FindUserByPhone(formattedPhone)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// New user - needs to complete profile
			return true, nil
		}
		return false, err
	}

	// Existing user - user is not nil, so return false (not new user)
	_ = user
	return false, nil
}

func (s *Service) CompleteProfile(phoneNumber string, req *CompleteProfileRequest) (*AuthResponse, error) {
	formattedPhone := util.FormatPhoneNumber(phoneNumber)

	// Check if user already exists
	existingUser, err := s.repo.FindUserByPhone(formattedPhone)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	var user *User

	if existingUser == nil {
		// Create new user
		user = &User{
			PhoneNumber:     formattedPhone,
			FullName:        req.FullName,
			IsActive:        true,
			IsPhoneVerified: true,
		}
		if err := s.repo.CreateUser(user); err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}
	} else {
		// Update existing user
		user = existingUser
		user.FullName = req.FullName
		if err := s.repo.UpdateUser(user); err != nil {
			return nil, fmt.Errorf("failed to update user: %w", err)
		}
	}

	// Create user roles
	for _, roleType := range req.Roles {
		userRole := &UserRole{
			UserID:   user.ID,
			RoleType: roleType,
			IsActive: true,
		}
		if err := s.repo.CreateUserRole(userRole); err != nil {
			// Skip if role already exists (unique constraint violation)
			continue
		}
	}

	// Get all user roles
	roles, err := s.repo.GetUserRoles(user.ID)
	if err != nil {
		return nil, err
	}

	// Generate JWT token
	token, err := util.GenerateJWT(user.ID, roles, s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &AuthResponse{
		Token: token,
		User: &UserData{
			ID:              user.ID.String(),
			PhoneNumber:     user.PhoneNumber,
			FullName:        user.FullName,
			ProfilePhotoURL: user.ProfilePhotoURL,
			Roles:           roles,
		},
	}, nil
}

func (s *Service) Login(phoneNumber string) (*AuthResponse, error) {
	formattedPhone := util.FormatPhoneNumber(phoneNumber)

	user, err := s.repo.FindUserByPhone(formattedPhone)
	if err != nil {
		return nil, errors.New("user not found")
	}

	roles, err := s.repo.GetUserRoles(user.ID)
	if err != nil {
		return nil, err
	}

	token, err := util.GenerateJWT(user.ID, roles, s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &AuthResponse{
		Token: token,
		User: &UserData{
			ID:              user.ID.String(),
			PhoneNumber:     user.PhoneNumber,
			FullName:        user.FullName,
			ProfilePhotoURL: user.ProfilePhotoURL,
			Roles:           roles,
		},
	}, nil
}

func (s *Service) LoginWithOTP(phoneNumber, otpCode string) (*AuthResponse, error) {
	formattedPhone := util.FormatPhoneNumber(phoneNumber)

	// Verify OTP
	otp, err := s.repo.FindValidOTP(formattedPhone, otpCode)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("invalid or expired OTP")
		}
		return nil, err
	}

	if otp.Attempts >= 3 {
		return nil, errors.New("OTP attempt limit exceeded")
	}

	// Mark OTP as used
	if err := s.repo.MarkOTPAsUsed(otp.ID); err != nil {
		return nil, err
	}

	// Find user
	user, err := s.repo.FindUserByPhone(formattedPhone)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found, please complete registration first")
		}
		return nil, err
	}

	// Get user roles
	roles, err := s.repo.GetUserRoles(user.ID)
	if err != nil {
		return nil, err
	}

	if len(roles) == 0 {
		return nil, errors.New("user has no roles assigned, please contact support")
	}

	// Generate JWT token with roles
	token, err := util.GenerateJWT(user.ID, roles, s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &AuthResponse{
		Token: token,
		User: &UserData{
			ID:              user.ID.String(),
			PhoneNumber:     user.PhoneNumber,
			FullName:        user.FullName,
			ProfilePhotoURL: user.ProfilePhotoURL,
			Roles:           roles,
		},
	}, nil
}

func generateOTPCode() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%06d", rand.Intn(1000000))
}