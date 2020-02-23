package echo

import (
	"testing"

	"github.com/fox-one/pkg/uuid"
	"github.com/stretchr/testify/assert"
)

func TestAuth(t *testing.T) {
	sessionID := uuid.New()
	conversationID := uuid.New()
	userID := uuid.New()

	token, err := SignToken(userID, sessionID, conversationID)
	if err != nil {
		t.Error(err)
		return
	}

	if cid, err := ParseToken(token, sessionID); err != nil {
		t.Error(err)
	} else {
		assert.Equal(t, conversationID, cid)
	}
}
