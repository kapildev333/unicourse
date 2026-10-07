package grade

import (
	"encoding/json"
	"time"
)

type FinalGrade struct {
	UserID         string      `json:"userId"`
	Email          string      `json:"email"`
	DisplayName    string      `json:"displayName"`
	Status         string      `json:"status"`
	Percent        json.Number `json:"percent,omitempty"`
	Letter         string      `json:"letter,omitempty"`
	Points         json.Number `json:"points,omitempty"`
	Credits        json.Number `json:"credits"`
	OverrideReason string      `json:"overrideReason,omitempty"`
	FinalizedBy    string      `json:"finalizedBy"`
	FinalizedAt    time.Time   `json:"finalizedAt"`
	id             string      // row id, the override audit entity
}
