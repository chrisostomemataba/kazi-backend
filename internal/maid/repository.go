package maid

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateMaidProfile(ctx context.Context, profile *MaidProfile) error {
	return r.db.WithContext(ctx).Create(profile).Error
}

func (r *Repository) GetMaidProfileByUserID(ctx context.Context, userID uuid.UUID) (*MaidProfile, error) {
	var profile MaidProfile
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&profile).Error
	return &profile, err
}

func (r *Repository) UpdateMaidProfile(ctx context.Context, profile *MaidProfile) error {
	return r.db.WithContext(ctx).Save(profile).Error
}

func (r *Repository) CreateMaidService(ctx context.Context, service *MaidService) error {
	return r.db.WithContext(ctx).Create(service).Error
}

func (r *Repository) DeleteMaidServices(ctx context.Context, maidID uuid.UUID) error {
	return r.db.WithContext(ctx).Where("maid_id = ?", maidID).Delete(&MaidService{}).Error
}

func (r *Repository) GetMaidServices(ctx context.Context, maidID uuid.UUID) ([]string, error) {
	var services []string
	err := r.db.WithContext(ctx).
		Model(&MaidService{}).
		Where("maid_id = ?", maidID).
		Pluck("service_type", &services).Error
	return services, err
}

func (r *Repository) CreateVerificationDocument(ctx context.Context, doc *MaidVerificationDocument) error {
	return r.db.WithContext(ctx).Create(doc).Error
}

func (r *Repository) GetVerificationDocuments(ctx context.Context, maidID uuid.UUID) ([]MaidVerificationDocument, error) {
	var docs []MaidVerificationDocument
	err := r.db.WithContext(ctx).Where("maid_id = ?", maidID).Find(&docs).Error
	return docs, err
}

func (r *Repository) GetMaidStatistics(ctx context.Context, maidID uuid.UUID) (*MaidStatistics, error) {
	var stats MaidStatistics
	err := r.db.WithContext(ctx).Where("maid_id = ?", maidID).First(&stats).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Create default stats if not exists
			stats = MaidStatistics{
				MaidID:           maidID,
				AverageRating:    0.0,
				TotalReviews:     0,
				TotalJobsCompleted: 0,
			}
			r.db.WithContext(ctx).Create(&stats)
			return &stats, nil
		}
		return nil, err
	}
	return &stats, nil
}

func (r *Repository) SearchMaids(ctx context.Context, req *SearchMaidsRequest) ([]MaidSearchResult, error) {
	query := r.db.WithContext(ctx).
		Table("users u").
		Select(`
			u.id,
			u.full_name,
			u.profile_photo_url,
			mp.gender,
			mp.district,
			mp.ward,
			mp.hourly_rate,
			mp.offers_contracts,
			mp.monthly_contract_rate,
			COALESCE(ms.average_rating, 0) as average_rating,
			COALESCE(ms.total_reviews, 0) as total_reviews,
			COALESCE(ms.total_jobs_completed, 0) as total_jobs_completed
		`).
		Joins("JOIN maid_profiles mp ON u.id = mp.user_id").
		Joins("LEFT JOIN maid_statistics ms ON u.id = ms.maid_id").
		Where("mp.verification_status = ?", "approved").
		Where("mp.is_available_now = ?", true)

	// Location-based search (Haversine formula)
	if req.Latitude != nil && req.Longitude != nil {
		radiusKM := req.RadiusKM
		if radiusKM == 0 {
			radiusKM = 10 // default 10km
		}
		
		query = query.Select(fmt.Sprintf(`
			u.id,
			u.full_name,
			u.profile_photo_url,
			mp.gender,
			mp.district,
			mp.ward,
			mp.hourly_rate,
			mp.offers_contracts,
			mp.monthly_contract_rate,
			COALESCE(ms.average_rating, 0) as average_rating,
			COALESCE(ms.total_reviews, 0) as total_reviews,
			COALESCE(ms.total_jobs_completed, 0) as total_jobs_completed,
			(
				6371 * acos(
					cos(radians(%f)) * 
					cos(radians(mp.home_location_lat)) * 
					cos(radians(mp.home_location_lng) - radians(%f)) + 
					sin(radians(%f)) * 
					sin(radians(mp.home_location_lat))
				)
			) AS distance_km
		`, *req.Latitude, *req.Longitude, *req.Latitude)).
		Having("distance_km <= ?", radiusKM).
		Order("distance_km ASC, ms.average_rating DESC")
	} else {
		query = query.Order("ms.average_rating DESC, ms.total_reviews DESC")
	}

	// Filters
	if req.ServiceType != "" {
		query = query.Where("EXISTS (SELECT 1 FROM maid_services WHERE maid_id = u.id AND service_type = ?)", req.ServiceType)
	}

	if req.MinHourlyRate != nil {
		query = query.Where("mp.hourly_rate >= ?", *req.MinHourlyRate)
	}

	if req.MaxHourlyRate != nil {
		query = query.Where("mp.hourly_rate <= ?", *req.MaxHourlyRate)
	}

	if req.OffersContracts != nil {
		query = query.Where("mp.offers_contracts = ?", *req.OffersContracts)
	}

	if req.Gender != "" {
		query = query.Where("mp.gender = ?", req.Gender)
	}

	if req.MinRating != nil {
		query = query.Where("COALESCE(ms.average_rating, 0) >= ?", *req.MinRating)
	}

	// Pagination
	page := req.Page
	if page < 1 {
		page = 1
	}
	limit := req.Limit
	if limit < 1 || limit > 50 {
		limit = 20
	}
	offset := (page - 1) * limit

	query = query.Limit(limit).Offset(offset)

	var results []MaidSearchResult
	err := query.Find(&results).Error
	if err != nil {
		return nil, err
	}

	// Fetch services for each maid
	for i := range results {
		maidID, _ := uuid.Parse(results[i].ID)
		services, _ := r.GetMaidServices(ctx, maidID)
		results[i].Services = services
	}

	return results, nil
}

func (r *Repository) GetPendingVerifications(ctx context.Context, limit, offset int) ([]MaidProfile, error) {
	var profiles []MaidProfile
	err := r.db.WithContext(ctx).
		Where("verification_status = ?", "pending").
		Order("created_at ASC").
		Limit(limit).
		Offset(offset).
		Find(&profiles).Error
	return profiles, err
}

func (r *Repository) UpdateVerificationStatus(ctx context.Context, maidID uuid.UUID, status, reason string) error {
	updates := map[string]interface{}{
		"verification_status": status,
		"rejection_reason":    reason,
	}
	if status == "approved" {
		updates["verified_at"] = gorm.Expr("NOW()")
	}
	return r.db.WithContext(ctx).
		Model(&MaidProfile{}).
		Where("user_id = ?", maidID).
		Updates(updates).Error
}

func (r *Repository) GetOrCreateWallet(ctx context.Context, maidID uuid.UUID) (*MaidWallet, error) {
	var wallet MaidWallet
	err := r.db.WithContext(ctx).Where("maid_id = ?", maidID).First(&wallet).Error
	if err == gorm.ErrRecordNotFound {
		wallet = MaidWallet{MaidID: maidID}
		if createErr := r.db.WithContext(ctx).Create(&wallet).Error; createErr != nil {
			return nil, createErr
		}
		return &wallet, nil
	}
	return &wallet, err
}
 
func (r *Repository) GetWalletTransactions(ctx context.Context, maidID uuid.UUID, limit, offset int) ([]WalletTransaction, error) {
	var txs []WalletTransaction
	err := r.db.WithContext(ctx).
		Where("maid_id = ?", maidID).
		Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&txs).Error
	return txs, err
}

// GetPendingEscrowAmount sums payouts for jobs the customer has paid for but
// which have not yet been released to the maid — shown as "pending" on the wallet.
func (r *Repository) GetPendingEscrowAmount(ctx context.Context, maidID uuid.UUID) (int, error) {
	var pendingAmount int
	err := r.db.WithContext(ctx).
		Table("bookings").
		Select("COALESCE(SUM(booking_pricings.maid_payout_amount), 0)").
		Joins("JOIN booking_pricings ON booking_pricings.booking_id = bookings.id").
		Where("bookings.maid_id = ? AND bookings.payment_status IN ?",
			maidID, []string{"paid_held_escrow", "disbursement_pending"}).
		Scan(&pendingAmount).Error
	return pendingAmount, err
}