package characters

import (
	"context"
	"fmt"
	"sort"
	"time"

	"guardian-tracker/bungie-service/cache"
	"guardian-tracker/bungie-service/services/bungie"
)

// bungieBaseURL prefixes the relative icon paths Bungie returns so the frontend
// can use them directly without knowing the CDN host.
const bungieBaseURL = "https://www.bungie.net"

// Service fetches and shapes a user's Destiny 2 characters.
type Service struct {
	bungieClient *bungie.Client
	cache        cache.Cache
	cacheTTL     time.Duration
}

// NewService creates a new characters service.
func NewService(bungieClient *bungie.Client, c cache.Cache, cacheTTL time.Duration) *Service {
	return &Service{
		bungieClient: bungieClient,
		cache:        c,
		cacheTTL:     cacheTTL,
	}
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

// GetCharacters returns the user's characters, most-recently-played first.
func (s *Service) GetCharacters(ctx context.Context, membershipType int, membershipID, accessToken string) ([]Character, error) {
	cacheKey := fmt.Sprintf("characters:%d:%s", membershipType, membershipID)
	if cached, found := s.cache.Get(cacheKey); found {
		if chars, ok := cached.([]Character); ok {
			return chars, nil
		}
	}

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

	// Most recently played first. Bungie returns ISO-8601 UTC timestamps, so a
	// lexicographic comparison orders them chronologically.
	sort.Slice(chars, func(i, j int) bool {
		return chars[i].DateLastPlayed > chars[j].DateLastPlayed
	})

	s.cache.Set(cacheKey, chars, s.cacheTTL)
	return chars, nil
}

// InvalidateCache drops the cached character list for a user.
func (s *Service) InvalidateCache(membershipType int, membershipID string) {
	s.cache.Delete(fmt.Sprintf("characters:%d:%s", membershipType, membershipID))
}

func absoluteURL(path string) string {
	if path == "" {
		return ""
	}
	return bungieBaseURL + path
}
