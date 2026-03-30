//go:build ignore
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"kazi-backend/config"
	"kazi-backend/internal/auth"
	"kazi-backend/internal/common/database"
	"kazi-backend/internal/maid"

)

type SeedMaid struct {
	FullName            string
	PhoneNumber         string
	Gender              string
	DateOfBirth         string
	Bio                 string
	HomeAddress         string
	Latitude            float64
	Longitude           float64
	District            string
	Ward                string
	HourlyRate          int
	OffersContracts     bool
	MonthlyContractRate int
	Services            []string
	IDNumber            string
	IDType              string
}

var seedMaids = []SeedMaid{
	{
		FullName:            "Mary John Mushi",
		PhoneNumber:         "255712345001",
		Gender:              "female",
		DateOfBirth:         "1990-05-15",
		Bio:                 "Mtaalamu wa usafi na upishi na uzoefu wa miaka 5. Ninafanya kazi kwa bidii na ninapenda usafi sana.",
		HomeAddress:         "House 123, Mbezi Beach, Kinondoni",
		Latitude:            -6.7537,
		Longitude:           39.2323,
		District:            "Kinondoni",
		Ward:                "Mbezi",
		HourlyRate:          8000,
		OffersContracts:     true,
		MonthlyContractRate: 300000,
		Services:            []string{"cleaning", "cooking"},
		IDNumber:            "19900515-12345-67890-12",
		IDType:              "NIDA",
	},
	{
		FullName:            "Neema Saidi Hassan",
		PhoneNumber:         "255712345002",
		Gender:              "female",
		DateOfBirth:         "1988-08-20",
		Bio:                 "Nina uzoefu wa miaka 7 katika usafi wa nyumba na ufuaji. Mwaminifu na mwenye nidhamu.",
		HomeAddress:         "House 456, Mikocheni, Kinondoni",
		Latitude:            -6.7724,
		Longitude:           39.2531,
		District:            "Kinondoni",
		Ward:                "Mikocheni",
		HourlyRate:          12000,
		OffersContracts:     true,
		MonthlyContractRate: 450000,
		Services:            []string{"cleaning", "laundry", "ironing"},
		IDNumber:            "19880820-23456-78901-23",
		IDType:              "NIDA",
	},
	{
		FullName:            "Amiri Joseph Kamara",
		PhoneNumber:         "255712345003",
		Gender:              "male",
		DateOfBirth:         "1992-03-10",
		Bio:                 "Mpishi hodari na mtaalamu wa vyakula vya Kiswahili na Kihindi. Uzoefu wa miaka 6.",
		HomeAddress:         "House 789, Masaki, Kinondoni",
		Latitude:            -6.7650,
		Longitude:           39.2700,
		District:            "Kinondoni",
		Ward:                "Msasani",
		HourlyRate:          15000,
		OffersContracts:     false,
		MonthlyContractRate: 0,
		Services:            []string{"cooking"},
		IDNumber:            "19920310-34567-89012-34",
		IDType:              "NIDA",
	},
	{
		FullName:            "Zainabu Ahmed Ally",
		PhoneNumber:         "255712345004",
		Gender:              "female",
		DateOfBirth:         "1995-11-25",
		Bio:                 "Mtaalamu wa kazi za nyumbani. Ninaweza kufanya usafi, upishi na kutunza watoto. Mwenye subira.",
		HomeAddress:         "House 321, Sinza, Kinondoni",
		Latitude:            -6.7800,
		Longitude:           39.2450,
		District:            "Kinondoni",
		Ward:                "Sinza",
		HourlyRate:          9000,
		OffersContracts:     true,
		MonthlyContractRate: 350000,
		Services:            []string{"cleaning", "cooking", "childcare"},
		IDNumber:            "19951125-45678-90123-45",
		IDType:              "NIDA",
	},
	{
		FullName:            "Mariam John Mwita",
		PhoneNumber:         "255712345005",
		Gender:              "female",
		DateOfBirth:         "1987-07-12",
		Bio:                 "Mtaalamu wa usafi wa nyumba kubwa na ofisi. Uzoefu wa miaka 10. Mwaminifu na mwenye ujuzi.",
		HomeAddress:         "House 555, Oysterbay, Kinondoni",
		Latitude:            -6.7700,
		Longitude:           39.2650,
		District:            "Kinondoni",
		Ward:                "Msasani",
		HourlyRate:          10000,
		OffersContracts:     false,
		MonthlyContractRate: 0,
		Services:            []string{"cleaning", "laundry"},
		IDNumber:            "19870712-56789-01234-56",
		IDType:              "NIDA",
	},
	{
		FullName:            "Bakari Rashid Omar",
		PhoneNumber:         "255712345006",
		Gender:              "male",
		DateOfBirth:         "1991-02-18",
		Bio:                 "Fundi wa bustani na usafi wa nje ya nyumba. Uzoefu wa miaka 4. Ninafanya kazi kwa bidii.",
		HomeAddress:         "House 777, Tegeta, Kinondoni",
		Latitude:            -6.6900,
		Longitude:           39.2300,
		District:            "Kinondoni",
		Ward:                "Tegeta",
		HourlyRate:          7000,
		OffersContracts:     true,
		MonthlyContractRate: 280000,
		Services:            []string{"cleaning"},
		IDNumber:            "19910218-67890-12345-67",
		IDType:              "NIDA",
	},
	{
		FullName:            "Ester Mwakasege Juma",
		PhoneNumber:         "255712345007",
		Gender:              "female",
		DateOfBirth:         "1993-09-05",
		Bio:                 "Mtaalamu wa kutunza watoto na upishi. Mwenye upendo na subira na watoto. Uzoefu wa miaka 5.",
		HomeAddress:         "House 888, Mwenge, Kinondoni",
		Latitude:            -6.7600,
		Longitude:           39.2400,
		District:            "Kinondoni",
		Ward:                "Mwenge",
		HourlyRate:          11000,
		OffersContracts:     true,
		MonthlyContractRate: 400000,
		Services:            []string{"childcare", "cooking"},
		IDNumber:            "19930905-78901-23456-78",
		IDType:              "NIDA",
	},
	{
		FullName:            "Grace Peter Ndunguru",
		PhoneNumber:         "255712345008",
		Gender:              "female",
		DateOfBirth:         "1989-12-30",
		Bio:                 "Mtaalamu wa ufuaji na upiga pasi. Uzoefu wa miaka 8. Ninafanya kazi kwa ufundi.",
		HomeAddress:         "House 999, Kawe, Kinondoni",
		Latitude:            -6.7400,
		Longitude:           39.2200,
		District:            "Kinondoni",
		Ward:                "Kawe",
		HourlyRate:          8500,
		OffersContracts:     false,
		MonthlyContractRate: 0,
		Services:            []string{"laundry", "ironing"},
		IDNumber:            "19891230-89012-34567-89",
		IDType:              "NIDA",
	},
}

func main() {
	cfg := config.LoadConfig()

	db, err := database.Connect(cfg.DatabaseURL, true)
	if err != nil {
		log.Fatal("Database connection failed:", err)
	}

	ctx := context.Background()

	fmt.Println("🌱 Starting maid seeding process...")

	for i, seedData := range seedMaids {
		fmt.Printf("\n[%d/%d] Creating maid: %s\n", i+1, len(seedMaids), seedData.FullName)

		// Create user
		user := &auth.User{
			PhoneNumber:     seedData.PhoneNumber,
			FullName:        seedData.FullName,
			IsActive:        true,
			IsPhoneVerified: true,
		}

		if err := db.WithContext(ctx).Create(user).Error; err != nil {
			log.Printf("❌ Failed to create user: %v", err)
			continue
		}

		// Create user role
		userRole := &auth.UserRole{
			UserID:   user.ID,
			RoleType: "maid",
			IsActive: true,
		}

		if err := db.WithContext(ctx).Create(userRole).Error; err != nil {
			log.Printf("❌ Failed to create role: %v", err)
			continue
		}

		// Parse date of birth
		dob, _ := time.Parse("2006-01-02", seedData.DateOfBirth)

		// Create maid profile
		var monthlyRate *int
		if seedData.OffersContracts {
			monthlyRate = &seedData.MonthlyContractRate
		}

		maidProfile := &maid.MaidProfile{
			UserID:              user.ID,
			Bio:                 seedData.Bio,
			Gender:              seedData.Gender,
			DateOfBirth:         &dob,
			HomeAddress:         seedData.HomeAddress,
			HomeLocationLat:     &seedData.Latitude,
			HomeLocationLng:     &seedData.Longitude,
			District:            seedData.District,
			Ward:                seedData.Ward,
			HourlyRate:          seedData.HourlyRate,
			OffersContracts:     seedData.OffersContracts,
			MonthlyContractRate: monthlyRate,
			VerificationStatus:  "approved", // Auto-approve for seed data
			IDNumber:            seedData.IDNumber,
			IDType:              seedData.IDType,
			IsAvailableNow:      true,
		}

		verifiedAt := time.Now()
		maidProfile.VerifiedAt = &verifiedAt

		if err := db.WithContext(ctx).Create(maidProfile).Error; err != nil {
			log.Printf("❌ Failed to create maid profile: %v", err)
			continue
		}

		// Create maid services
		for _, serviceType := range seedData.Services {
			maidService := &maid.MaidService{
				MaidID:      user.ID,
				ServiceType: serviceType,
			}
			db.WithContext(ctx).Create(maidService)
		}

		// Create maid statistics
		stats := &maid.MaidStatistics{
			MaidID:                  user.ID,
			AverageRating:           4.8,
			TotalReviews:            124,
			TotalJobsCompleted:      247,
			TotalContractsCompleted: 12,
			TotalEarnings:           5600000,
		}
		db.WithContext(ctx).Create(stats)

		fmt.Printf("✅ Successfully created maid: %s (ID: %s)\n", seedData.FullName, user.ID)
	}

	fmt.Println("\n🎉 Seeding completed!")
	fmt.Printf("📊 Total maids created: %d\n", len(seedMaids))
}