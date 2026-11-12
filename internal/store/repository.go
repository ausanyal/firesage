package store

import (
	"context"
	"time"

	"github.com/ausanyal/firesage/internal/engine"
)

// SavedPlan represents a versioned user FIRE scenario document.
// In V1, plans are persisted Local-First in the browser's IndexedDB and exported/imported as JSON.
// This schema and PlanRepository interface provide a zero-refactor seam for adding
// Google Sign-In (Firebase Auth OIDC) + Google Cloud Firestore synchronization.
type SavedPlan struct {
	ID            string           `json:"id"`
	OwnerUID      string           `json:"owner_uid,omitempty"` // Populated when Google Sign-In is active
	Name          string           `json:"name"`
	SchemaVersion int              `json:"schema_version"`
	CreatedAt     time.Time        `json:"created_at"`
	UpdatedAt     time.Time        `json:"updated_at"`
	Input         engine.PlanInput `json:"input"`
}

// PlanRepository defines the cloud persistence contract for future Firestore / Cloud SQL adapters.
type PlanRepository interface {
	ListPlans(ctx context.Context, ownerUID string) ([]SavedPlan, error)
	GetPlan(ctx context.Context, ownerUID string, planID string) (*SavedPlan, error)
	UpsertPlan(ctx context.Context, plan SavedPlan) error
	DeletePlan(ctx context.Context, ownerUID string, planID string) error
}
