package database

import (
	"log"
	"os"
	"time"

	"kikundibora/models"

	"golang.org/x/crypto/bcrypt"
)

func Seed() {
	var userCount int64
	DB.Model(&models.User{}).Count(&userCount)
	if userCount == 0 {
		log.Println("Seeding users...")
		seedUsers()
	} else {
		log.Println("Users already exist — skipping user seed")
	}

	ensureAdmin()
	seedMembers()

	log.Println("Seed complete")
}

func seedUsers() {
	hashed, _ := bcrypt.GenerateFromPassword([]byte("demo123"), bcrypt.DefaultCost)
	pwd := string(hashed)

	demoEmail := func(e string) *string { return &e }

	users := []models.User{
		{Name: "Mwenyekiti Juma", Email: demoEmail("juma@kikundi.tz"), Phone: "0710000001", Password: pwd, Role: models.RoleChair, Status: models.UserStatusActive, MustChangePassword: false, IsActive: true},
		{Name: "Hazina Fatuma", Email: demoEmail("fatuma@kikundi.tz"), Phone: "0710000002", Password: pwd, Role: models.RoleTreasurer, Status: models.UserStatusActive, MustChangePassword: false, IsActive: true},
		{Name: "Katibu Rashidi", Email: demoEmail("rashidi@kikundi.tz"), Phone: "0710000003", Password: pwd, Role: models.RoleSecretary, Status: models.UserStatusActive, MustChangePassword: false, IsActive: true},
		{Name: "Asha Mwakalinga", Email: demoEmail("asha@kikundi.tz"), Phone: "0710000004", Password: pwd, Role: models.RoleMember, Status: models.UserStatusActive, MustChangePassword: false, IsActive: true},
	}

	for i := range users {
		DB.Create(&users[i])
	}

	positions := []models.UserPosition{
		{UserID: users[0].ID, PositionType: models.PositionChairperson, IsActive: true},
		{UserID: users[1].ID, PositionType: models.PositionTreasurer, IsActive: true},
		{UserID: users[2].ID, PositionType: models.PositionSecretary, IsActive: true},
	}
	for i := range positions {
		DB.Create(&positions[i])
	}
}

func seedMembers() {
	var memberCount int64
	DB.Model(&models.Member{}).Where("deleted_at IS NULL").Count(&memberCount)
	if memberCount > 0 {
		log.Printf("Members already exist (%d) — skipping member seed", memberCount)
		return
	}

	log.Println("Seeding members...")

	var chair models.User
	if err := DB.Where("role = ?", models.RoleChair).First(&chair).Error; err != nil {
		log.Println("No chair user found — skipping member seed")
		return
	}

	date := func(s string) time.Time {
		t, _ := time.Parse("2006-01-02", s)
		return t
	}

	members := []models.Member{
		{MemberNo: "KKK-0001", FullName: "Asha Mwakalinga", Phone: "0712345678", IsActive: true, JoinedAt: date("2024-01-15"), RegisteredBy: chair.ID},
		{MemberNo: "KKK-0002", FullName: "Juma Kibwana", Phone: "0754321890", IsActive: true, JoinedAt: date("2024-02-10"), RegisteredBy: chair.ID},
		{MemberNo: "KKK-0003", FullName: "Neema Mhagama", Phone: "0689112233", IsActive: true, JoinedAt: date("2024-03-05"), RegisteredBy: chair.ID},
		{MemberNo: "KKK-0004", FullName: "Hamisi Mtenga", Phone: "0765998877", IsActive: true, JoinedAt: date("2024-04-20"), RegisteredBy: chair.ID},
		{MemberNo: "KKK-0005", FullName: "Rehema Sanga", Phone: "0784556677", IsActive: true, JoinedAt: date("2024-05-12"), RegisteredBy: chair.ID},
	}

	for i := range members {
		DB.Create(&members[i])
	}

	log.Printf("Seeded %d members", len(members))
}

func ensureAdmin() {
	// SECURITY: Read admin password from environment variable
	// Never hardcode passwords in source code
	adminPwd := os.Getenv("ADMIN_PASSWORD")
	if adminPwd == "" {
		log.Println("WARNING: ADMIN_PASSWORD not set. Skipping admin account creation.")
		log.Println("Set ADMIN_PASSWORD environment variable to create the admin account.")
		return
	}

	if len(adminPwd) < 8 {
		log.Println("WARNING: ADMIN_PASSWORD must be at least 8 characters. Skipping admin account creation.")
		return
	}

	adminHashed, err := bcrypt.GenerateFromPassword([]byte(adminPwd), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("ERROR: Failed to hash admin password: %v", err)
		return
	}
	adminPwdHash := string(adminHashed)

	var user models.User
	err = DB.Where("role = ? AND phone = ?", models.RoleAdmin, "0000000000").First(&user).Error
	if err != nil {
		// Admin doesn't exist — create it
		log.Println("Creating admin account...")
		admin := models.User{
			Name:               "System Admin",
			Phone:              "0000000000",
			Password:           adminPwdHash,
			Role:               models.RoleAdmin,
			Status:             models.UserStatusActive,
			MustChangePassword: false,
			IsActive:           true,
		}
		DB.Create(&admin)
		log.Println("Admin account created — phone: 0000000000")
		return
	}

	// Admin exists — update password and ensure no forced password change
	DB.Model(&user).Updates(map[string]interface{}{
		"password":              adminPwdHash,
		"must_change_password":  false,
	})
	log.Println("Admin account updated")
}
