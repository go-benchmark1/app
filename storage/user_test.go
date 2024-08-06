package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUser(t *testing.T) {
	db := openTestDb()
	store := From(db)

	//Test get admin user
	user, err := store.GetUser(1)
	assert.Nil(t, err)
	assert.Equal(t, user.Username, "admin")

	//Update user test
	user.Username = "foo"

	err = store.UpdateUser(user)
	assert.Nil(t, err)

	user, err = store.GetUser(1)
	assert.Nil(t, err)
	assert.Equal(t, user.Username, "foo")

	// Test get user by username
	_, err = store.GetActiveUserByUsername("foo")
	assert.Nil(t, err)
}
