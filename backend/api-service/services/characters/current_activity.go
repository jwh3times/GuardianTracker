package characters

import (
	"context"
	"errors"
	"fmt"
	"time"

	"guardian-tracker/api-service/services/bungie"
)

// CurrentActivityState keeps the four verified component-204 outcomes apart.
// Idle is only a returned zero activity hash; anything Bungie did not return is
// unavailable, never idle.
type CurrentActivityState string

const (
	CurrentActivityReady       CurrentActivityState = "ready"
	CurrentActivityIdle        CurrentActivityState = "idle"
	CurrentActivityUnavailable CurrentActivityState = "unavailable"
	CurrentActivityUnknown     CurrentActivityState = "unknown"
)

// CurrentActivity is the owner-only, best-effort current activity of one
// Guardian. It deliberately carries no membership, character, instance, or
// definition identifiers, and no start time: dateActivityStarted establishes
// no freshness guarantee. Names are present only when they resolved.
type CurrentActivity struct {
	State        CurrentActivityState `json:"state"`
	ActivityName string               `json:"activityName,omitempty"`
	ModeName     string               `json:"modeName,omitempty"`
	PlaylistName string               `json:"playlistName,omitempty"`
	FetchedAt    time.Time            `json:"fetchedAt"`
}

// GetCurrentActivity returns the component-204 current activity for a current
// character in the authenticated membership. Roster validation happens before
// the component-204 request so an arbitrary character ID cannot cross the
// Bungie seam.
func (s *Service) GetCurrentActivity(ctx context.Context, membershipType int, membershipID, characterID, accessToken string) (*CurrentActivity, error) {
	characters, err := s.GetCharacters(ctx, membershipType, membershipID, accessToken)
	if err != nil {
		return nil, fmt.Errorf("validate current-activity character: %w", err)
	}
	found := false
	for _, character := range characters {
		if character.CharacterID == characterID {
			found = true
			break
		}
	}
	if !found {
		return nil, ErrCharacterNotFound
	}

	resp, err := s.bungieClient.GetProfile(ctx, membershipType, membershipID, accessToken, []int{bungie.ComponentCharacterActivities})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch character current activity: %w", err)
	}

	current := &CurrentActivity{State: CurrentActivityUnavailable, FetchedAt: time.Now().UTC()}
	activities := resp.Response.CharacterActivities
	if !componentAvailable(activities.Data, activities.Disabled) {
		return current, nil
	}
	entry, ok := (*activities.Data)[characterID]
	if !ok || entry.CurrentActivityHash == nil {
		return current, nil
	}
	if *entry.CurrentActivityHash == 0 {
		current.State = CurrentActivityIdle
		return current, nil
	}

	if s.activityFacts == nil {
		return nil, errors.New("characters: activity definition reader is unavailable")
	}
	activityHashes := []uint32{*entry.CurrentActivityHash}
	if hash := populated(entry.CurrentPlaylistActivityHash); hash != 0 {
		activityHashes = append(activityHashes, hash)
	}
	definitions, err := s.activityFacts.GetActivityDefinitions(activityHashes)
	if err != nil {
		return nil, fmt.Errorf("resolve current activity: %w", err)
	}

	current.State = CurrentActivityUnknown
	if definition, ok := definitions[*entry.CurrentActivityHash]; ok && definition != nil {
		current.State = CurrentActivityReady
		current.ActivityName = definition.DisplayProperties.Name
		if current.ActivityName == "" {
			current.ActivityName = "Unnamed activity"
		}
	}
	if hash := populated(entry.CurrentPlaylistActivityHash); hash != 0 {
		if definition, ok := definitions[hash]; ok && definition != nil {
			current.PlaylistName = definition.DisplayProperties.Name
		}
	}

	if hash := populated(entry.CurrentActivityModeHash); hash != 0 {
		modes, err := s.activityFacts.GetActivityModeDefinitions([]uint32{hash})
		if err != nil {
			return nil, fmt.Errorf("resolve current activity mode: %w", err)
		}
		if mode, ok := modes[hash]; ok && mode != nil {
			current.ModeName = mode.DisplayProperties.Name
		}
	}
	return current, nil
}

// populated reads an optional component-204 hash, treating absent as zero.
func populated(hash *uint32) uint32 {
	if hash == nil {
		return 0
	}
	return *hash
}
