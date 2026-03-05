package main

import (
	"context"
	"fmt"
	"log"

	"kazi-backend/config"
	"kazi-backend/internal/admin"
	"kazi-backend/internal/common/database"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	cfg := config.LoadConfig()

	db, err := database.Connect(cfg.DatabaseURL, true)
	if err != nil {
		log.Fatal("Database connection failed:", err)
	}

	// Create default admin user
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal("Failed to hash password:", err)
	}

	adminUser := &admin.AdminUser{
		Username:     "admin",
		PasswordHash: string(hashedPassword),
		FullName:     "System Administrator",
		Role:         "super_admin",
		IsActive:     true,
	}

	if err := db.WithContext(context.Background()).Create(adminUser).Error; err != nil {
		log.Fatal("Failed to create admin user:", err)
	}

	fmt.Println("✅ Admin user created successfully!")
	fmt.Println("Username: admin")
	fmt.Println("Password: admin123")
	fmt.Println("⚠️  Please change the password after first login")
}