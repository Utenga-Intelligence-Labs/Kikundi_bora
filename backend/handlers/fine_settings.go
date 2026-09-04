package handlers

import (
	"kikundibora/database"
	"kikundibora/models"
)

func applyFineProposal(g models.Group, proposal *models.GroupSettingProposal) error {
	settings, err := database.GetOrCreateFineSettings(g.ID)
	if err != nil {
		return err
	}
	if proposal.FinesEnabled != nil {
		settings.Enabled = *proposal.FinesEnabled
	}
	if proposal.FineType != nil && *proposal.FineType != "" {
		settings.FineType = *proposal.FineType
	}
	settings.FineAmount = proposal.FineAmount
	settings.FinePercentage = proposal.FinePercentage
	if proposal.GracePeriodDays != nil {
		settings.GracePeriodDays = *proposal.GracePeriodDays
	}
	return database.DB.Save(settings).Error
}
