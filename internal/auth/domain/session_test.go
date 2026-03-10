package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSessionTableName(t *testing.T) {
	var s Session
	require.Equal(t, "user_sessions", s.TableName())
}

func TestSessionBeforeCreateSetsAuditFields(t *testing.T) {
	s := &Session{}

	err := s.BeforeCreate(nil)
	require.NoError(t, err)

	require.NotEmpty(t, s.ID)
	require.False(t, s.CreatedAt.IsZero())
	require.False(t, s.UpdatedAt.IsZero())
	require.False(t, s.LastUsedAt.IsZero())
	require.WithinDuration(t, s.CreatedAt, s.UpdatedAt, time.Second)
}

func TestSessionBeforeUpdateRefreshesUpdatedAt(t *testing.T) {
	old := time.Now().UTC().Add(-1 * time.Hour)
	s := &Session{UpdatedAt: old}

	err := s.BeforeUpdate(nil)
	require.NoError(t, err)

	require.True(t, s.UpdatedAt.After(old))
}
