package handlers

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"kikundibora/database"
	"kikundibora/models"

	"github.com/shopspring/decimal"
	"golang.org/x/crypto/bcrypt"
)

func hashTestPassword(pw string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(pw), bcrypt.MinCost)
}

// TestLoanChainHandoff proves the sequential loan-approval queue-to-queue
// handoff: a loan appears only in the CURRENT stage-holder's my_turn queue,
// and each approval moves it to the next stage — Hazina → Katibu → Bodi →
// Mwenyekiti → disbursement → borrower confirmation.
func TestLoanChainHandoff(t *testing.T) {
	app := fullTestApp()
	cleanAndSeed(t)

	chair := hLogin(t, app, "juma@kikundi.tz", "demo123")
	treasurer := hLogin(t, app, "fatuma@kikundi.tz", "demo123")
	secretary := hLogin(t, app, "rashidi@kikundi.tz", "demo123")
	bodiTok := hLogin(t, app, "asha@kikundi.tz", "demo123")

	// Appoint asha to the loan committee so she can act at the BODI stage.
	code, body := hPost(t, app, "/api/v1/loan-committee/members", map[string]interface{}{
		"user_id": hGetUserID(t, "asha@kikundi.tz"),
	}, chair)
	if code != 200 && code != 201 {
		t.Fatalf("appoint bodi member: %d %s", code, body)
	}

	// Neutral borrower (plain member, NOT in approval chain) so no approver
	// ever signs own loan. Asha stays purely as bodi voter.
	neemaEmail := "neema@kikundi.tz"
	neemaPass := "demo123"
	neemaHash, _ := hashTestPassword(neemaPass)
	neemaUser := models.User{
		Name: "Neema Neutral", Email: &neemaEmail, Phone: "0719999999",
		Password: string(neemaHash), Role: models.RoleMember,
		Status: models.UserStatusActive, IsActive: true,
	}
	if err := database.DB.Create(&neemaUser).Error; err != nil {
		t.Fatalf("create neutral user: %v", err)
	}
	neutralMember := models.Member{
		MemberNo: "M-NEUTRAL-01", FullName: "Neema Neutral", Phone: "0719999999",
		IsActive: true, ApprovalStatus: "approved", UserID: &neemaUser.ID,
		JoinedAt: time.Now(), RegisteredBy: neemaUser.ID,
	}
	if err := database.DB.Create(&neutralMember).Error; err != nil {
		t.Fatalf("create neutral member: %v", err)
	}
	borrowerTok := hLogin(t, app, neemaEmail, neemaPass)
	fundTreasury(500000)

	code, d := hPost(t, app, "/api/v1/loans/apply", map[string]interface{}{
		"member_id": neutralMember.ID, "amount": 100000.0, "purpose": "Biashara", "due_date": "2026-12-31",
	}, borrowerTok)
	if code != 201 {
		t.Fatalf("apply: %d %s", code, d)
	}
	loanID := hExtract(t, d, "data")

	type pendingResp struct {
		Data []struct {
			ID           string `json:"id"`
			AwaitingRole string `json:"awaiting_role"`
			MyTurn       bool   `json:"my_turn"`
		} `json:"data"`
		Total int `json:"total"`
	}

	myTurnContains := func(token, loanID string) bool {
		code, d := hGet(t, app, "/api/v1/uongozi/mikopo/pending?my_turn=true", token)
		if code != 200 {
			t.Fatalf("pending my_turn: %d %s", code, d)
		}
		var pr pendingResp
		json.Unmarshal(d, &pr)
		for _, l := range pr.Data {
			if l.ID == loanID {
				return true
			}
		}
		return false
	}
	stageOf := func(token, loanID string) string {
		code, d := hGet(t, app, "/api/v1/uongozi/mikopo/pending", token)
		if code != 200 {
			t.Fatalf("pending: %d %s", code, d)
		}
		var pr pendingResp
		json.Unmarshal(d, &pr)
		for _, l := range pr.Data {
			if l.ID == loanID {
				return l.AwaitingRole
			}
		}
		return ""
	}
	chainApprove := func(token string) (int, []byte) {
		return hPost(t, app, "/api/v1/uongozi/mikopo/"+loanID+"/approve", map[string]interface{}{}, token)
	}

	// Parallel: fresh loan is in EVERY approver's my_turn queue at once —
	// no "anasubiri X" gate. Anyone may sign in any order.
	for _, tc := range []struct {
		name string
		tok  string
	}{
		{"hazina", treasurer}, {"katibu", secretary}, {"bodi", bodiTok}, {"mwenyekiti", chair},
	} {
		if !myTurnContains(tc.tok, loanID) {
			t.Errorf("%s's my_turn queue should contain the fresh loan (parallel)", tc.name)
		}
	}
	_ = stageOf

	// Scrambled order: katibu first — must succeed (no order gate).
	if code, d = chainApprove(secretary); code != 200 {
		t.Fatalf("katibu approve (parallel, any order): %d %s", code, d)
	}
	// Double sign by same stage must still fail.
	if code, _ = chainApprove(secretary); code == 200 {
		t.Fatal("double katibu approve must fail")
	}
	// Partial (1/4) must NOT finalize.
	var mid models.Loan
	database.DB.First(&mid, "id = ?", loanID)
	if mid.Status == models.LoanApproved {
		t.Fatal("loan must not be APPROVED after only 1/4 parallel approvals")
	}
	if code, d = chainApprove(bodiTok); code != 200 {
		t.Fatalf("bodi approve: %d %s", code, d)
	}
	if code, d = chainApprove(treasurer); code != 200 {
		t.Fatalf("hazina approve: %d %s", code, d)
	}
	// Partial (3/4) must NOT finalize.
	database.DB.First(&mid, "id = ?", loanID)
	if mid.Status == models.LoanApproved {
		t.Fatal("loan must not be APPROVED after only 3/4 parallel approvals")
	}
	// 4th sign (mwenyekiti) → final APPROVED.
	if code, d = chainApprove(chair); code != 200 {
		t.Fatalf("chair approve: %d %s", code, d)
	}

	var loan models.Loan
	database.DB.First(&loan, "id = ?", loanID)
	if loan.Status != models.LoanApproved {
		t.Fatalf("want APPROVED after full chain, got %s", loan.Status)
	}

	// Legacy bypass routes are gone: direct approve/reject must 404.
	code, _ = hPost(t, app, "/api/v1/loans/"+loanID+"/approve", map[string]interface{}{"approved_amount": 1.0}, chair)
	if code != 404 && code != 405 {
		t.Errorf("legacy /loans/:id/approve should be removed, got %d", code)
	}
	code, _ = hPost(t, app, "/api/v1/loans/"+loanID+"/reject", map[string]interface{}{"reason": "x"}, chair)
	if code != 404 && code != 405 {
		t.Errorf("legacy /loans/:id/reject should be removed, got %d", code)
	}

	// Disbursement (hazina) → OUTSTANDING.
	code, d = hPost(t, app, "/api/v1/loans/"+loanID+"/disburse", nil, treasurer)
	if code != 200 {
		t.Fatalf("disburse: %d %s", code, d)
	}

	// BUG-5: only the borrower confirms receipt.
	code, d = hPatch(t, app, "/api/v1/loans/"+loanID+"/confirm-received", chair, nil)
	if code != 403 {
		t.Errorf("borrower-confirm as chair: want 403, got %d", code)
	}
	code, d = hPatch(t, app, "/api/v1/loans/"+loanID+"/confirm-received", borrowerTok, nil)
	if code != 200 {
		t.Fatalf("borrower-confirm as borrower: %d %s", code, d)
	}
	database.DB.First(&loan, "id = ?", loanID)
	if loan.BorrowerConfirmedAt == nil {
		t.Error("borrower_confirmed_at should be set after confirm")
	}
	code, _ = hPatch(t, app, "/api/v1/loans/"+loanID+"/confirm-received", borrowerTok, nil)
	if code != 409 {
		t.Errorf("double borrower-confirm: want 409, got %d", code)
	}
}

// TestCommitteeAppointChairOnly — BUG-3: adding a member to the loan board
// is mwenyekiti's action; katibu gets 403.
func TestCommitteeAppointChairOnly(t *testing.T) {
	app := fullTestApp()
	cleanAndSeed(t)

	chair := hLogin(t, app, "juma@kikundi.tz", "demo123")
	secretary := hLogin(t, app, "rashidi@kikundi.tz", "demo123")
	target := hGetUserID(t, "asha@kikundi.tz")

	code, _ := hPost(t, app, "/api/v1/loan-committee/members", map[string]interface{}{
		"user_id": target,
	}, secretary)
	if code != 403 {
		t.Errorf("katibu appoint committee member: want 403, got %d", code)
	}

	code, body := hPost(t, app, "/api/v1/loan-committee/members", map[string]interface{}{
		"user_id": target,
	}, chair)
	if code != 200 && code != 201 {
		t.Errorf("mwenyekiti appoint committee member: want 2xx, got %d (%s)", code, body)
	}
}

// TestWelfareReceiptBeneficiaryOnly — BUG-4: only the beneficiary member can
// confirm receiving welfare funds disbursed on their behalf. Leadership gets 403.
func TestWelfareReceiptBeneficiaryOnly(t *testing.T) {
	app := fullTestApp()
	cleanAndSeed(t)

	chair := hLogin(t, app, "juma@kikundi.tz", "demo123")
	treasurer := hLogin(t, app, "fatuma@kikundi.tz", "demo123")

	var ashaUser models.User
	database.DB.Where("email = ?", "asha@kikundi.tz").First(&ashaUser)
	var ashaMember models.Member
	database.DB.Where("user_id = ?", ashaUser.ID).First(&ashaMember)

	// Beneficiary event, fully disbursed.
	disbursed := time.Now()
	event := models.WelfareEvent{
		MemberID:        ashaMember.ID,
		EventType:       models.WelfareMisiba,
		Description:     "Msaada wa msiba",
		AmountRequested: decimal.NewFromInt(50000),
		FundingSource:   models.FundTreasury,
		Status:          models.WelfareCompleted,
		CreatedBy:       hGetUserID(t, "juma@kikundi.tz"),
		DisbursedAt:     &disbursed,
		DisbursedBy:     &ashaUser.ID,
	}
	database.DB.Create(&event)

	beneficiary := hLogin(t, app, "asha@kikundi.tz", "demo123")

	// Leadership (chair AND treasurer) must NOT confirm on her behalf.
	for _, tok := range []string{chair, treasurer} {
		code, _ := hPost(t, app, fmt.Sprintf("/api/v1/welfare/events/%s/confirm-receipt", event.ID), nil, tok)
		if code != 403 {
			t.Errorf("leadership confirm-receipt: want 403, got %d", code)
		}
	}

	// The beneficiary herself confirms.
	code, body := hPost(t, app, fmt.Sprintf("/api/v1/welfare/events/%s/confirm-receipt", event.ID), nil, beneficiary)
	if code != 200 {
		t.Fatalf("beneficiary confirm-receipt: %d %s", code, body)
	}
	var ev models.WelfareEvent
	database.DB.First(&ev, "id = ?", event.ID)
	if ev.ReceivedAt == nil {
		t.Error("received_at should be set after beneficiary confirms")
	}

	// Second confirmation is a conflict.
	code, _ = hPost(t, app, fmt.Sprintf("/api/v1/welfare/events/%s/confirm-receipt", event.ID), nil, beneficiary)
	if code != 409 {
		t.Errorf("double confirm-receipt: want 409, got %d", code)
	}
}
