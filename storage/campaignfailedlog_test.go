package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mailbadger/app/entities"
)

func TestCampaignFailedLog(t *testing.T) {
	db := openTestDb()
	store := From(db)

	campaign1 := &entities.Campaign{
		Model:        entities.Model{ID: 1},
		UserID:       1,
		Name:         "bla",
		TemplateID:   0,
		BaseTemplate: nil,
		Status:       "draft",
	}
	err := store.CreateCampaign(campaign1)
	assert.Nil(t, err)

	err = store.LogFailedCampaign(campaign1, "asd")
	assert.Nil(t, err)

	c, err := store.GetCampaign(campaign1.ID, campaign1.UserID)
	assert.Nil(t, err)
	assert.Equal(t, "failed", c.Status)
}
