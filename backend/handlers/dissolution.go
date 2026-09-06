package handlers

import (
	"strings"
	"time"

	"kikundibora/database"
	"kikundibora/middleware"
	"kikundibora/models"
	"kikundibora/services"

	"github.com/gofiber/fiber/v2"
	"github.com/shopspring/decimal"
)

type DissolutionHandler struct{}

func NewDissolutionHandler() *DissolutionHandler { return &DissolutionHandler{} }

// POST /api/v1/groups/:id/dissolution-proposals  (mwenyekiti/katibu)
func (h *DissolutionHandler) Propose(c *fiber.Ctx) error {
	groupID := c.Params("id")
	if ok, _ := database.IsCurrentGroup(groupID); !ok {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Kikundi hakijapatikana"})
	}
	var g models.Group
	if err := database.DB.First(&g, "id = ?", groupID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Kikundi hakijapatikana"})
	}
	if g.Status == "dissolved" {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"message": "Kikundi tayari kimevunjwa"})
	}
	// block if voting_open exists
	var open int64
	database.DB.Model(&models.GroupDissolutionProposal{}).Where("group_id = ? AND status = ?", models.DissolutionVotingOpen, models.DissolutionVotingOpen).Count(&open)
	if open > 0 {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"message": "Kuna pendekezo la uvunjaji linaloendelea kupigiwa kura"})
	}
	var req models.CreateDissolutionProposalRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Data si sahihi"})
	}
	if req.CycleSpanYears != 1 && req.CycleSpanYears != 2 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "cycle_span_years lazima iwe 1 au 2"})
	}
	deadline, err := time.Parse(time.RFC3339, req.VotingDeadline)
	if err != nil {
		deadline, err = time.Parse("2006-01-02", req.VotingDeadline)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "voting_deadline si sahihi (tumie YYYY-MM-DD au RFC3339)"})
		}
		deadline = deadline.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
	}
	if deadline.Before(time.Now()) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "voting_deadline lazima iwe baadaye"})
	}
	now := time.Now()
	periodEnd := dateOnly(now)
	periodStart := periodEnd.AddDate(-req.CycleSpanYears, 0, 0)

	prop := models.GroupDissolutionProposal{
		GroupID:        groupID,
		ProposedBy:     middleware.GetUserID(c),
		CycleSpanYears: req.CycleSpanYears,
		PeriodStart:    periodStart,
		PeriodEnd:      periodEnd,
		Status:         models.DissolutionVotingOpen,
		VotingDeadline: deadline,
	}
	if err := database.DB.Create(&prop).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kutengeneza pendekezo"})
	}
	userID := middleware.GetUserID(c)
	services.LogAudit(c, &userID, models.AuditCreate, "group_dissolution_proposals", &prop.ID, nil, map[string]interface{}{"group_id": groupID, "span": req.CycleSpanYears})
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "Pendekezo la uvunjaji limetengenezwa", "data": prop})
}

// POST /api/v1/dissolution-proposals/:id/vote  (any approved member, one vote - second vote updates)
func (h *DissolutionHandler) Vote(c *fiber.Ctx) error {
	propID := c.Params("id")
	var prop models.GroupDissolutionProposal
	if err := database.DB.First(&prop, "id = ?", propID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Pendekezo halijapatikana"})
	}
	if prop.Status != models.DissolutionVotingOpen {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Kura imefungwa"})
	}
	if time.Now().After(prop.VotingDeadline) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Muda wa kupiga kura umeisha"})
	}
	userID := middleware.GetUserID(c)
	var member models.Member
	if err := database.DB.Where("user_id = ? AND deleted_at IS NULL AND approval_status = 'approved' AND is_active = TRUE", userID).First(&member).Error; err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"message": "Ni wanachama walioidhinishwa pekee wanaoweza kupiga kura"})
	}
	var req models.VoteRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Data si sahihi"})
	}
	vote := strings.ToLower(strings.TrimSpace(req.Vote))
	if vote != "yes" && vote != "no" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "vote lazima iwe yes au no"})
	}
	var existing models.DissolutionVote
	err := database.DB.Where("proposal_id = ? AND member_id = ?", propID, member.ID).First(&existing).Error
	if err == nil {
		existing.Vote = vote
		database.DB.Save(&existing)
		return c.JSON(fiber.Map{"message": "Kura imesasishwa", "data": existing})
	}
	v := models.DissolutionVote{ProposalID: propID, MemberID: member.ID, Vote: vote}
	if err := database.DB.Create(&v).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kuhifadhi kura"})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "Kura imehifadhiwa", "data": v})
}

// GET /api/v1/dissolution-proposals/:id
func (h *DissolutionHandler) Get(c *fiber.Ctx) error {
	propID := c.Params("id")
	var prop models.GroupDissolutionProposal
	if err := database.DB.First(&prop, "id = ?", propID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Pendekezo halijapatikana"})
	}
	var yes, no, total int64
	database.DB.Model(&models.DissolutionVote{}).Where("proposal_id = ? AND vote = ?", propID, "yes").Count(&yes)
	database.DB.Model(&models.DissolutionVote{}).Where("proposal_id = ? AND vote = ?", propID, "no").Count(&no)
	database.DB.Model(&models.DissolutionVote{}).Where("proposal_id = ?", propID).Count(&total)
	// my vote
	var myVote *string
	if uid := middleware.GetUserID(c); uid != "" {
		var m models.Member
		if err := database.DB.Where("user_id = ? AND deleted_at IS NULL", uid).First(&m).Error; err == nil {
			var v models.DissolutionVote
			if err := database.DB.Where("proposal_id = ? AND member_id = ?", propID, m.ID).First(&v).Error; err == nil {
				myVote = &v.Vote
			}
		}
	}
	approved := yes > no && total > 0 // simple majority of votes cast
	return c.JSON(fiber.Map{
		"data": prop,
		"tally": fiber.Map{"yes": yes, "no": no, "total": total, "approved": approved},
		"my_vote": myVote,
	})
}

// GET /api/v1/groups/:id/dissolution-proposals  list
func (h *DissolutionHandler) ListByGroup(c *fiber.Ctx) error {
	groupID := c.Params("id")
	var props []models.GroupDissolutionProposal
	database.DB.Where("group_id = ?", groupID).Order("created_at DESC").Find(&props)
	return c.JSON(fiber.Map{"data": props})
}

// POST /api/v1/dissolution-proposals/:id/execute  (mwenyekiti/katibu after deadline + threshold)
func (h *DissolutionHandler) Execute(c *fiber.Ctx) error {
	propID := c.Params("id")
	var prop models.GroupDissolutionProposal
	if err := database.DB.First(&prop, "id = ?", propID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Pendekezo halijapatikana"})
	}
	if prop.Status == models.DissolutionExecuted {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"message": "Tayari imetekelezwa"})
	}
	if prop.Status != models.DissolutionVotingOpen {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Pendekezo si voting_open"})
	}
	if time.Now().Before(prop.VotingDeadline) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Muda wa kupiga kura bado haujaisha"})
	}
	var yes, no, total int64
	database.DB.Model(&models.DissolutionVote{}).Where("proposal_id = ? AND vote = ?", propID, "yes").Count(&yes)
	database.DB.Model(&models.DissolutionVote{}).Where("proposal_id = ? AND vote = ?", propID, "no").Count(&no)
	database.DB.Model(&models.DissolutionVote{}).Where("proposal_id = ?", propID).Count(&total)
	if !(yes > no && total > 0) {
		// mark rejected
		database.DB.Model(&prop).Update("status", models.DissolutionRejected)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Kura hazikupita — simple majority inahitajika (>50% yes)", "tally": fiber.Map{"yes": yes, "no": no, "total": total}})
	}
	// calculate share-out: principal-only, netted against obligations, visible itemized
	if err := h.executePayouts(&prop); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kutekeleza: " + err.Error()})
	}
	now := time.Now()
	database.DB.Model(&prop).Updates(map[string]interface{}{"status": models.DissolutionExecuted, "executed_at": now})
	database.DB.Model(&models.Group{}).Where("id = ?", prop.GroupID).Update("status", "dissolved")
	userID := middleware.GetUserID(c)
	services.LogAudit(c, &userID, models.AuditUpdate, "group_dissolution_proposals", &prop.ID, map[string]interface{}{"status": models.DissolutionVotingOpen}, map[string]interface{}{"status": models.DissolutionExecuted})
	return c.JSON(fiber.Map{"message": "Uvunjaji umetekelezwa — payouts zimetengenezwa", "data": prop})
}

func (h *DissolutionHandler) executePayouts(prop *models.GroupDissolutionProposal) error {
	var members []models.Member
	database.DB.Where("deleted_at IS NULL AND is_active = TRUE AND approval_status = 'approved'").Find(&members)
	for _, m := range members {
		contributed := sumMainContributions(m.ID, prop.PeriodStart, prop.PeriodEnd)
		owed := totalOwedForMember(prop.GroupID, m.ID)
		net := contributed.Sub(owed)
		if net.LessThan(decimal.Zero) {
			net = decimal.Zero
		}
		p := models.DissolutionPayout{
			ProposalID:       prop.ID,
			MemberID:         m.ID,
			TotalContributed: contributed,
			TotalOwed:        owed,
			AmountOwed:       net,
			Status:           "pending",
			CalculatedAt:     time.Now(),
		}
		if err := database.DB.Create(&p).Error; err != nil {
			return err
		}
	}
	return nil
}

func sumMainContributions(memberID string, start, end time.Time) decimal.Decimal {
	// Sum of PAID contributions within period (reconciled truth source — includes confirmed member contributions)
	type r struct{ Total string }
	var row r
	database.DB.Raw(`SELECT COALESCE(SUM(amount),0)::text AS total FROM contributions
		WHERE member_id = ? AND status = 'PAID' AND paid_at >= ? AND paid_at <= ?`,
		memberID, start.Format("2006-01-02"), end.Format("2006-01-02")).Scan(&row)
	d, _ := decimal.NewFromString(row.Total)
	return d
}

func totalOwedForMember(groupID, memberID string) decimal.Decimal {
	mo, err := services.GetMemberObligations(groupID, memberID, time.Now())
	if err != nil {
		return decimal.Zero
	}
	return mo.GrandTotal
}

func dateOnly(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func IsGroupDissolved() bool {
	var g models.Group
	if err := database.DB.First(&g).Error; err != nil {
		return false
	}
	return g.Status == "dissolved"
}

// GET /api/v1/dissolution-proposals/:id/payouts
func (h *DissolutionHandler) ListPayouts(c *fiber.Ctx) error {
	propID := c.Params("id")
	var payouts []models.DissolutionPayout
	database.DB.Where("proposal_id = ?", propID).Preload("Member").Order("created_at ASC").Find(&payouts)
	return c.JSON(fiber.Map{"data": payouts})
}

// PATCH /api/v1/dissolution-payouts/:id/mark-paid  (mweka hazina)
func (h *DissolutionHandler) MarkPaid(c *fiber.Ctx) error {
	id := c.Params("id")
	var p models.DissolutionPayout
	if err := database.DB.First(&p, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Payout haijapatikana"})
	}
	if p.Status == "paid" {
		return c.JSON(fiber.Map{"message": "Tayari imelipwa", "data": p})
	}
	now := time.Now()
	uid := middleware.GetUserID(c)
	p.Status = "paid"
	p.PaidAt = &now
	p.PaidBy = &uid
	database.DB.Save(&p)
	services.LogAudit(c, &uid, models.AuditUpdate, "dissolution_payouts", &p.ID, map[string]interface{}{"status": "pending"}, map[string]interface{}{"status": "paid"})
	return c.JSON(fiber.Map{"message": "Imethibitishwa kulipwa", "data": p})
}

// GET /api/v1/dissolution-payouts/me  - my payouts
func (h *DissolutionHandler) MyPayouts(c *fiber.Ctx) error {
	uid := middleware.GetUserID(c)
	var m models.Member
	if err := database.DB.Where("user_id = ? AND deleted_at IS NULL", uid).First(&m).Error; err != nil {
		return c.JSON(fiber.Map{"data": []models.DissolutionPayout{}})
	}
	var payouts []models.DissolutionPayout
	database.DB.Where("member_id = ?", m.ID).Order("calculated_at DESC").Find(&payouts)
	return c.JSON(fiber.Map{"data": payouts})
}
