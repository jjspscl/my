package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMagicToken_GeneratesToken(t *testing.T) {
	token, err := NewMagicToken("user@test.com", 15*time.Minute)
	require.NoError(t, err)

	assert.NotEmpty(t, token.Token)
	assert.Equal(t, "user@test.com", token.Email)
	assert.False(t, token.ExpiresAt.IsZero())
	assert.Nil(t, token.UsedAt)
}

func TestNewMagicToken_TokenIsHexEncoded(t *testing.T) {
	token, err := NewMagicToken("user@test.com", 15*time.Minute)
	require.NoError(t, err)

	// 32 bytes = 64 hex chars
	assert.Len(t, token.Token, 64)
}

func TestNewMagicToken_ExpiryIsInFuture(t *testing.T) {
	token, err := NewMagicToken("user@test.com", 15*time.Minute)
	require.NoError(t, err)

	assert.True(t, token.ExpiresAt.After(time.Now().UTC()))
}

func TestNewMagicToken_ExpiryIsCorrectDuration(t *testing.T) {
	ttl := 10 * time.Minute
	token, err := NewMagicToken("user@test.com", ttl)
	require.NoError(t, err)

	expected := time.Now().UTC().Add(ttl)
	// Allow 2 second tolerance for test execution time
	assert.InDelta(t, expected.Unix(), token.ExpiresAt.Unix(), 2)
}

func TestIsExpired_FreshToken_ReturnsFalse(t *testing.T) {
	token, err := NewMagicToken("user@test.com", 15*time.Minute)
	require.NoError(t, err)

	assert.False(t, token.IsExpired())
}

func TestIsExpired_ExpiredToken_ReturnsTrue(t *testing.T) {
	token, err := NewMagicToken("user@test.com", 15*time.Minute)
	require.NoError(t, err)

	// Manually set expiry to the past
	token.ExpiresAt = time.Now().UTC().Add(-1 * time.Minute)

	assert.True(t, token.IsExpired())
}

func TestIsUsed_NotUsed_ReturnsFalse(t *testing.T) {
	token, err := NewMagicToken("user@test.com", 15*time.Minute)
	require.NoError(t, err)

	assert.False(t, token.IsUsed())
}

func TestIsUsed_AfterMarkUsed_ReturnsTrue(t *testing.T) {
	token, err := NewMagicToken("user@test.com", 15*time.Minute)
	require.NoError(t, err)

	token.MarkUsed()
	assert.True(t, token.IsUsed())
	assert.NotNil(t, token.UsedAt)
}

func TestMarkUsed_SetsUsedAt(t *testing.T) {
	token, err := NewMagicToken("user@test.com", 15*time.Minute)
	require.NoError(t, err)

	before := time.Now().UTC()
	token.MarkUsed()
	after := time.Now().UTC()

	assert.NotNil(t, token.UsedAt)
	assert.True(t, token.UsedAt.After(before.Add(-1*time.Second)))
	assert.True(t, token.UsedAt.Before(after.Add(1*time.Second)))
}
