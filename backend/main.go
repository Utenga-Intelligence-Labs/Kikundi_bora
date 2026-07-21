package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"kikundibora/config"
	"kikundibora/database"
	"kikundibora/handlers"
	"kikundibora/middleware"
	"kikundibora/models"
	"kikundibora/services"

	"github.com/gofiber/fiber/v2"
)

func main() {
	migrateFlag := flag.Bool("migrate", false, "Run database migration and seed, then exit")
	seedFlag := flag.Bool("seed", false, "Run seed only (no table drop), then exit")
	flag.Parse()

	config.Load()
	database.Connect()
	services.InitEmail()

	if *migrateFlag {
		database.AutoMigrate()
		database.Seed()
		log.Println("Migration complete. Exiting.")
		os.Exit(0)
	}

	if *seedFlag {
		database.Seed()
		log.Println("Seed complete. Exiting.")
		os.Exit(0)
	}

	app := fiber.New(fiber.Config{
		AppName:      "Kikundi API v1.0",
		ErrorHandler: errorHandler,
	})

	app.Use(middleware.SetupCORS())
	app.Use(middleware.SecurityHeaders())
	app.Use(middleware.RequestLogger())

	// Serve uploaded files
	app.Static("/uploads", "./uploads")

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "service": "kikundi-api"})
	})

	api := app.Group("/api/v1")

	authHandler := handlers.NewAuthHandler()
	memberHandler := handlers.NewMemberHandler()
	contribHandler := handlers.NewContributionHandler()
	loanHandler := handlers.NewLoanHandler()
	repayHandler := handlers.NewRepaymentHandler()
	dashHandler := handlers.NewDashboardHandler()
	notifHandler := handlers.NewNotificationHandler()
	auditHandler := handlers.NewAuditHandler()
	committeeHandler := handlers.NewLoanCommitteeHandler()
	welfareHandler := handlers.NewWelfareHandler()
	userMgmtHandler := handlers.NewUserManagementHandler()
	adminHandler := handlers.NewAdminHandler()
	pendingActionHandler := handlers.NewPendingActionHandler()
	uploadHandler := handlers.NewUploadHandler(config.AppConfig.PublicBaseURL)
	backupHandler := handlers.NewBackupHandler()
	reportHandler := handlers.NewReportHandler()
	leadershipHandler := handlers.NewLeadershipHandler()
	memberContribHandler := handlers.NewMemberContributionHandler()
	announcementHandler := handlers.NewAnnouncementHandler()

	auth := api.Group("/auth")
	auth.Post("/login", authHandler.Login)

	protected := api.Group("")
	protected.Use(middleware.AuthRequired)

	protected.Get("/me", authHandler.Me)
	protected.Put("/me", authHandler.UpdateProfile)
	protected.Post("/auth/change-password", authHandler.ChangePassword)
	protected.Post("/auth/logout", authHandler.Logout)
	protected.Post("/auth/first-login-setup", authHandler.FirstLoginSetup)

	// File upload routes
	protected.Post("/upload/avatar", uploadHandler.UploadAvatar)
	protected.Post("/upload/doc", uploadHandler.UploadDoc)

	protected.Get("/dashboard", dashHandler.Summary)

	// User Management routes (Mwenyekiti creates, Katibu approves)
	users := protected.Group("/users")
	users.Post("/create", middleware.RequireRoles(models.RoleChair), userMgmtHandler.CreateUser)
	users.Get("/pending", middleware.RequireRoles(models.RoleSecretary), userMgmtHandler.ListPending)
	users.Get("/", middleware.RequireRoles(models.RoleChair, models.RoleSecretary), userMgmtHandler.ListUsers)
	users.Post("/:id/approve", middleware.RequireRoles(models.RoleSecretary), userMgmtHandler.ApproveUser)
	users.Post("/:id/reject", middleware.RequireRoles(models.RoleSecretary), userMgmtHandler.RejectUser)
	// Chair may reset non-admin user passwords (temp returned once); not full admin powers
	users.Post("/:id/reset-password", middleware.RequireRoles(models.RoleChair), userMgmtHandler.ResetUserPassword)

	members := protected.Group("/members")
	members.Get("/", memberHandler.List)
	members.Get("/:id", memberHandler.Get)
	members.Post("/", middleware.RequireRoles(models.RoleChair, models.RoleSecretary, models.RoleTreasurer), memberHandler.Create)
	members.Put("/:id", middleware.RequireRoles(models.RoleChair, models.RoleSecretary), memberHandler.Update)
	members.Delete("/:id", middleware.RequireRoles(models.RoleChair), memberHandler.Delete)

	contribs := protected.Group("/contributions")
	contribs.Get("/", contribHandler.List)
	contribs.Post("/", middleware.RequirePosition(models.PositionTreasurer), contribHandler.Create)
	contribs.Put("/:id", middleware.RequirePosition(models.PositionTreasurer), contribHandler.Edit)
	contribs.Get("/monthly-report", contribHandler.MonthlyReport)

	loans := protected.Group("/loans")
	loans.Get("/", loanHandler.List)
	loans.Get("/outstanding-report", loanHandler.OutstandingReport)
	loans.Get("/:id", loanHandler.Get)
	loans.Post("/apply", loanHandler.Apply)
	// Chair/treasurer can approve (legacy direct path); committee unanimous path also finalizes
	loans.Post("/:id/approve", middleware.RequireRoles(models.RoleChair, models.RoleTreasurer), loanHandler.Approve)
	loans.Post("/:id/reject", middleware.RequireRoles(models.RoleChair, models.RoleTreasurer), loanHandler.Reject)
	loans.Post("/:id/disburse", middleware.RequirePosition(models.PositionTreasurer), loanHandler.Disburse)

	repayments := protected.Group("/repayments")
	repayments.Get("/", repayHandler.List)
	repayments.Post("/", middleware.RequirePosition(models.PositionTreasurer), repayHandler.Record)

	notifs := protected.Group("/notifications")
	notifs.Get("/", notifHandler.List)
	notifs.Post("/read", notifHandler.MarkRead)

	protected.Get("/audit-logs", middleware.RequireRoles(models.RoleChair, models.RoleAdmin), auditHandler.List)
	protected.Get("/audit-logs/login-activity", middleware.RequireRoles(models.RoleChair, models.RoleAdmin), auditHandler.LoginActivity)
	protected.Get("/audit-logs/failed-logins", middleware.RequireRoles(models.RoleChair, models.RoleAdmin), auditHandler.FailedLogins)
	protected.Get("/audit-logs/summary", middleware.RequireRoles(models.RoleChair, models.RoleAdmin), auditHandler.AuditSummary)

	// Loan committee check endpoint (for frontend gating)
	protected.Get("/loan-committee/check", committeeHandler.IsCommitteeMember)

	// Loan committee routes
	committee := protected.Group("/loan-committee")
	committee.Use(middleware.RequireLoanCommitteeMember())

	committee.Get("/members", committeeHandler.ListMembers)
	committee.Post("/members", middleware.RequireRoles(models.RoleChair), committeeHandler.AppointMember)
	committee.Delete("/members/:id", middleware.RequireRoles(models.RoleChair), committeeHandler.RemoveMember)
	committee.Get("/loans", committeeHandler.ListLoans)
	committee.Get("/loans/:id", committeeHandler.GetLoan)
	committee.Post("/loans/:id/review", committeeHandler.SubmitReview)
	committee.Get("/dashboard", committeeHandler.GetDashboard)
	committee.Get("/history", committeeHandler.GetHistory)
	committee.Get("/report", committeeHandler.GetReport)
	committee.Get("/pending-count", committeeHandler.GetPendingLoansCount)

	// Welfare routes (Mfuko wa Kijamii)
	welfare := protected.Group("/welfare")

	// Dashboard — all authenticated users
	welfare.Get("/dashboard", welfareHandler.Dashboard)

	// Events — all can view, treasurer creates, chair OR secretary approves
	welfare.Get("/events", welfareHandler.ListEvents)
	welfare.Get("/events/:id", welfareHandler.GetEvent)
	welfare.Post("/events", middleware.RequireRoles(models.RoleTreasurer), welfareHandler.CreateEvent)
	welfare.Post("/events/:id/approve", middleware.RequireRoles(models.RoleChair, models.RoleSecretary), welfareHandler.ApproveEvent)
	welfare.Post("/events/:id/reject", middleware.RequireRoles(models.RoleChair, models.RoleSecretary), welfareHandler.RejectEvent)

	// Contributions — members see their own, treasurer/chair/secretary see all
	welfare.Get("/contributions", welfareHandler.ListContributions)
	welfare.Get("/my-contributions", welfareHandler.MyContributions)
	welfare.Post("/events/:id/contributions/:memberId/pay", middleware.RequireRoles(models.RoleTreasurer), welfareHandler.RecordPayment)
	welfare.Post("/events/:id/contributions/:memberId/waive", middleware.RequireRoles(models.RoleTreasurer), welfareHandler.WaiveContribution)

	// Disbursement — treasurer disburses after event is approved and fully funded
	welfare.Post("/events/:id/disburse", middleware.RequireRoles(models.RoleTreasurer), welfareHandler.DisburseEvent)

	// Pending Actions routes (Chairperson approves)
	pending := protected.Group("/pending-actions")
	pending.Get("/", middleware.RequirePosition(models.PositionChairperson), pendingActionHandler.List)
	pending.Get("/:id", middleware.RequirePosition(models.PositionChairperson), pendingActionHandler.Get)
	pending.Post("/:id/approve", middleware.RequirePosition(models.PositionChairperson), pendingActionHandler.Approve)
	pending.Post("/:id/reject", middleware.RequirePosition(models.PositionChairperson), pendingActionHandler.Reject)

	// Admin routes (Super Admin only — protected by RequireRoles middleware)
	admin := protected.Group("/admin")
	admin.Use(middleware.RequireRoles(models.RoleAdmin))
	admin.Get("/users", adminHandler.ListAllUsers)
	admin.Get("/logs", adminHandler.GetAdminLogs)
	admin.Post("/users/:id/override", adminHandler.OverrideUser)
	admin.Post("/users/:id/reset-password", adminHandler.ResetUserPassword)
	admin.Post("/auth/reset-password", authHandler.ResetPassword) // Admin-only password reset
	admin.Get("/health", adminHandler.GetSystemHealth)

	// Admin Backup routes
	admin.Post("/backup/generate", backupHandler.GenerateBackup)
	admin.Get("/backup/history", backupHandler.GetBackupHistory)
	admin.Get("/backup/settings", backupHandler.GetBackupSettings)
	admin.Post("/backup/settings", backupHandler.SaveBackupSettings)
	admin.Get("/backup/download/:id", backupHandler.DownloadBackup)

	// Reports routes (Chair only)
	reports := protected.Group("/reports")
	reports.Use(middleware.RequireRoles(models.RoleChair))
	reports.Get("/wanachama", reportHandler.MembersReport)
	reports.Get("/michango", reportHandler.ContributionsReport)
	reports.Get("/mikopo", reportHandler.LoansReport)
	reports.Get("/mapato", reportHandler.IncomeExpenseReport)
	reports.Get("/muhtasari", reportHandler.SummaryReport)

	// Member Contribution routes (Phase 4)
	michango := protected.Group("/michango")
	michango.Post("/", memberContribHandler.Submit)
	michango.Get("/mine", memberContribHandler.MyContributions)
	michango.Get("/pending", middleware.RequireLeadership(models.LeadershipChair, models.LeadershipTreasurer), memberContribHandler.PendingContributions)
	michango.Get("/", middleware.RequireLeadership(models.LeadershipChair, models.LeadershipTreasurer), memberContribHandler.AllContributions)
	michango.Post("/:id/confirm", middleware.RequireLeadership(models.LeadershipChair, models.LeadershipTreasurer), memberContribHandler.Confirm)
	michango.Post("/:id/reject", middleware.RequireLeadership(models.LeadershipChair, models.LeadershipTreasurer), memberContribHandler.Reject)

	// Leadership routes (dual plane — members with leadership roles)
	uongozi := protected.Group("/uongozi")
	uongozi.Use(middleware.RequireLeadership(models.LeadershipChair, models.LeadershipTreasurer, models.LeadershipSecretary))

	uongozi.Get("/dashboard", leadershipHandler.Dashboard)
	uongozi.Get("/quick-stats", leadershipHandler.QuickStats)
	uongozi.Post("/announcements", announcementHandler.Broadcast)
	uongozi.Get("/mikopo/pending", middleware.RequireLeadership(models.LeadershipChair, models.LeadershipTreasurer), leadershipHandler.PendingLoans)
	uongozi.Post("/mikopo/:id/approve", middleware.RequireLeadership(models.LeadershipChair, models.LeadershipTreasurer), leadershipHandler.ApproveLoan)
	uongozi.Get("/ripoti", leadershipHandler.Reports)
	uongozi.Get("/wanachama", memberHandler.List)

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("Shutting down gracefully...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := app.ShutdownWithContext(ctx); err != nil {
			log.Fatalf("Shutdown error: %v", err)
		}
		log.Println("Server stopped.")
	}()

	log.Println("Server starting on :" + config.AppConfig.Port)
	if err := app.Listen(":" + config.AppConfig.Port); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func errorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
	}
	log.Printf("ERROR [%d]: %v", code, err)
	message := "Hitilafu ya mfumo"
	if code < 500 {
		message = err.Error()
	}
	return c.Status(code).JSON(fiber.Map{"message": message})
}
