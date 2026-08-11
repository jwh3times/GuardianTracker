package characters

import (
	"context"
	"fmt"
	"sort"
	"time"

	"guardian-tracker/api-service/cache"
	"guardian-tracker/api-service/services/bungie"
)

const bungieBaseURL = "https://www.bungie.net"

// Service fetches and shapes a user's Destiny 2 characters.
type Service struct {
	bungieClient *bungie.Client
	cache        cache.Cache
	cacheTTL     time.Duration
}

func NewService(bungieClient *bungie.Client, c cache.Cache, cacheTTL time.Duration) *Service {
	return &Service{bungieClient: bungieClient, cache: c, cacheTTL: cacheTTL}
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
	return cache.Load(ctx, s.cache, charactersCacheKey(membershipType, membershipID), s.cacheTTL,
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

func (s *Service) InvalidateCache(membershipType int, membershipID string) {
	s.cache.Delete(fmt.Sprintf("characters:%d:%s", membershipType, membershipID))
}

func absoluteURL(path string) string {
	if path == "" {
		return ""
	}
	return bungieBaseURL + path
}
