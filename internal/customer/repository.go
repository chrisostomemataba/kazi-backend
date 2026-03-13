package customer

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetOrCreateProfile(ctx context.Context, userID uuid.UUID) (*CustomerProfile, error) {
	var profile CustomerProfile
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&profile).Error
	
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			profile = CustomerProfile{
				UserID:            userID,
				PreferredLanguage: "sw",
			}
			if err := r.db.WithContext(ctx).Create(&profile).Error; err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	
	return &profile, nil
}

func (r *Repository) AddLocation(ctx context.Context, location *CustomerLocation) error {
	// If this is set as default, unset all other defaults for this customer
	if location.IsDefault {
		r.db.WithContext(ctx).
			Model(&CustomerLocation{}).
			Where("customer_id = ?", location.CustomerID).
			Update("is_default", false)
	}
	
	return r.db.WithContext(ctx).Create(location).Error
}

func (r *Repository) GetCustomerLocations(ctx context.Context, customerID uuid.UUID) ([]CustomerLocation, error) {
	var locations []CustomerLocation
	err := r.db.WithContext(ctx).
		Where("customer_id = ?", customerID).
		Order("is_default DESC, created_at DESC").
		Find(&locations).Error
	return locations, err
}

func (r *Repository) DeleteLocation(ctx context.Context, locationID, customerID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("id = ? AND customer_id = ?", locationID, customerID).
		Delete(&CustomerLocation{}).Error
}

func (r *Repository) GetOrCreateStatistics(ctx context.Context, customerID uuid.UUID) (*CustomerStatistics, error) {
	var stats CustomerStatistics
	err := r.db.WithContext(ctx).Where("customer_id = ?", customerID).First(&stats).Error
	
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			stats = CustomerStatistics{
				CustomerID:        customerID,
				AverageMaidRating: 0.0,
				PaymentOnTimeRate: 100.0,
			}
			if err := r.db.WithContext(ctx).Create(&stats).Error; err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	
	return &stats, nil
}

func (r *Repository) UpdateStatistics(ctx context.Context, stats *CustomerStatistics) error {
	return r.db.WithContext(ctx).Save(stats).Error
}