package handlers

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"kikundibora/database"
	"kikundibora/models"

	"github.com/shopspring/decimal"
)

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

	// Asha's own member applies for a loan.
	var ashaUser models.User
	database.DB.Where("email = ?", "asha@kikundi.tz").First(&ashaUser)
	var ashaMember models.Member
	database.DB.Where("user_id = ?", ashaUser.ID).First(&ashaMember)
	fundTreasury(500000)

	code, d := hPost(t, app, "/api/v1/loans/apply", map[string]interface{}{
		"member_id": ashaMember.ID, "amount": 100000.0, "purpose": "Biashara", "due_date": "2026-12-31",
	}, bodiTok)
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

	// Stage 1: hazina's turn — visible ONLY to mweka hazina.
	if stage := stageOf(treasurer, loanID); stage != "hazina" {
		t.Fatalf("initial stage: want hazina, got %q", stage)
	}
	if !myTurnContains(treasurer, loanID) {
		t.Error("hazina's my_turn queue should contain the fresh loan")
	}
	if myTurnContains(secretary, loanID) {
		t.Error("katibu's my_turn queue must NOT contain the loan before hazina approves")
	}
	if myTurnContains(chair, loanID) {
		t.Error("mwenyekiti's my_turn queue must NOT contain the loan before hazina approves")
	}

	// Only the stage-holder can act.
	if code, _ = chainApprove(secretary); code == 200 {
		t.Fatal("katibu must not be able to approve at the hazina stage")
	}
	if code, d = chainApprove(treasurer); code != 200 {
		t.Fatalf("hazina approve: %d %s", code, d)
	}

	// Stage 2: katibu's turn — handoff proven (was NOT there before).
	if stage := stageOf(secretary, loanID); stage != "katibu" {
		t.Fatalf("after hazina: want katibu stage, got %q", stage)
	}
	if !myTurnContains(secretary, loanID) {
		t.Error("katibu's my_turn queue should contain the loan after hazina approves")
	}
	if myTurnContains(treasurer, loanID) {
		t.Error("hazina's my_turn queue must no longer contain the loan")
	}
	if code, _ = chainApprove(treasurer); code == 200 {
		t.Fatal("double hazina approve must fail")
	}
	if code, d = chainApprove(secretary); code != 200 {
		t.Fatalf("katibu approve: %d %s", code, d)
	}

	// Stage 3: bodi's turn.
	if !myTurnContains(bodiTok, loanID) {
		t.Error("bodi member's my_turn queue should contain the loan after katibu approves")
	}
	if myTurnContains(secretary, loanID) {
		t.Error("katibu's my_turn queue must no longer contain the loan after katibu approves")
	}
	if code, d = chainApprove(bodiTok); code != 200 {
		t.Fatalf("bodi approve: %d %s", code, d)
	}

	// Stage 4: mwenyekiti's turn → final APPROVED.
	if !myTurnContains(chair, loanID) {
		t.Error("mwenyekiti's my_turn queue should contain the loan after bodi approves")
	}
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
	code, d = hPatch(t, app, "/api/v1/loans/"+loanID+"/confirm-received", bodiTok, nil)
	if code != 200 {
		t.Fatalf("borrower-confirm as borrower: %d %s", code, d)
	}
	database.DB.First(&loan, "id = ?", loanID)
	if loan.BorrowerConfirmedAt == nil {
		t.Error("borrower_confirmed_at should be set after confirm")
	}
	code, _ = hPatch(t, app, "/api/v1/loans/"+loanID+"/confirm-received", bodiTok, nil)
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
