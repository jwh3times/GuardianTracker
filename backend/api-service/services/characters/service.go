package characters

import (
	"context"
	"fmt"
	"sort"
	"time"

	"guardian-tracker/api-service/cache"
	"guardian-tracker/api-service/services/bungie"
	"guardian-tracker/api-service/services/membershipstate"
)

const bungieBaseURL = "https://www.bungie.net"

// Service fetches and shapes a user's Destiny 2 characters.
type Service struct {
	bungieClient *bungie.Client
	cache        cache.Cache
	cacheTTL     time.Duration

	// refresh fences this owner's per-membership cache against a membership
	// data refresh: a roster loaded before the refresh may answer its own
	// request but may not be left behind afterwards (ADR 0018).
	refresh *membershipstate.Publication
}

func NewService(bungieClient *bungie.Client, c cache.Cache, cacheTTL time.Duration) *Service {
	s := &Service{bungieClient: bungieClient, cache: c, cacheTTL: cacheTTL}
	// The callback runs inside the publication's critical section, so it only
	// deletes the entry and never calls back into the publication.
	s.refresh = membershipstate.New(func(membershipType int, membershipID string) {
		if s.cache == nil {
			return
		}
		s.cache.Delete(charactersCacheKey(membershipType, membershipID))
	})
	return s
}

// Character is the frontend-facing representation of a Destiny 2 character.
type Character struct {
	CharacterID          string `json:"characterId"`
	ClassType            int    `json:"classType"`
	ClassName            string `json:"className"`
	RaceName             string `json:"raceName"`
	Light                int    `json:"light"`
	EmblemPath           string `json:"emblemPath"`
	EmblemBackgroundPath string `json:"emblemBackgroundPath"`
	DateLastPlayed       string `json:"dateLastPlayed"`
}

// charactersCacheKey holds a user's shaped character roster. Not scoped by
// manifest version: character components carry no manifest-resolved labels.
func charactersCacheKey(membershipType int, membershipID string) string {
	return fmt.Sprintf("characters:%d:%s", membershipType, membershipID)
}

// GetCharacters returns the user's characters sorted most-recently-played first.
func (s *Service) GetCharacters(ctx context.Context, membershipType int, membershipID, accessToken string) ([]Character, error) {
	return membershipstate.Load(ctx, s.refresh, s.cache, membershipType, membershipID,
		charactersCacheKey(membershipType, membershipID), s.cacheTTL,
		func() ([]Character, error) {
			resp, err := s.bungieClient.GetCharacters(ctx, membershipType, membershipID, accessToken)
			if err != nil {
				return nil, fmt.Errorf("failed to fetch characters: %w", err)
			}

			chars := make([]Character, 0, len(resp.Response.Characters.Data))
			for _, c := range resp.Response.Characters.Data {
				chars = append(chars, Character{
					CharacterID:          c.CharacterID,
					ClassType:            c.ClassType,
					ClassName:            bungie.GetClassName(c.ClassType),
					RaceName:             bungie.GetRaceName(c.RaceType),
					Light:                c.Light,
					EmblemPath:           absoluteURL(c.EmblemPath),
					EmblemBackgroundPath: absoluteURL(c.EmblemBackgroundPath),
					DateLastPlayed:       c.DateLastPlayed,
				})
			}

			// Bungie returns ISO-8601 UTC timestamps; lexicographic sort is chronological.
			sort.Slice(chars, func(i, j int) bool { return chars[i].DateLastPlayed > chars[j].DateLastPlayed })
			return chars, nil
		})
}

// InvalidateCache retires this owner's cached roster for one membership as a
// single transition: the generation advances and the entry is deleted together,
// so a load already in flight can still answer its own request but can no
// longer become reusable. Implements the Collections refresh participant
// contract.
func (s *Service) InvalidateCache(membershipType int, membershipID string) {
	s.refresh.Advance(membershipType, membershipID)
}

func absoluteURL(path string) string {
	if path == "" {
		return ""
	}
	return bungieBaseURL + path
}
