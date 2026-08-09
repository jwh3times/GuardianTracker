package weekly

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"guardian-tracker/api-service/cache"
	"guardian-tracker/api-service/observability"
	"guardian-tracker/api-service/services/bungie"
	"guardian-tracker/api-service/services/collections"
	"guardian-tracker/api-service/services/efficiency"
	"guardian-tracker/api-service/services/sources"
)

// Weekly is the assembled weekly recommendations payload sent to the frontend.
type Weekly struct {
	ResetLabel   string              `json:"resetLabel"`
	ResetIn      Duration            `json:"resetIn"`
	DailyResetIn Duration            `json:"dailyResetIn"`
	ResetAt      time.Time           `json:"resetAt"`            // next weekly reset (checkmark persistence key)
	FetchedAt    time.Time           `json:"fetchedAt"`          // when the underlying Bungie data was fetched (B8)
	Degraded     bool                `json:"degraded,omitempty"` // true when names/labels are placeholders (manifest still downloading)
	Xur          *Xur                `json:"xur"`
	Milestones   []Milestone         `json:"milestones"`
	Recommended  []RecommendedAction `json:"recommended"`
	DailyActions []DailyAction       `json:"dailyActions"`
}

// Duration is a decomposed time-until value.
type Duration struct {
	D int `json:"d,omitempty"`
	H int `json:"h,omitempty"`
	M int `json:"m,omitempty"`
}

// Xur holds Xûr's current inventory and schedule data.
type Xur struct {
	Present  bool      `json:"present"`
	LeavesIn Duration  `json:"leavesIn"`
	Location string    `json:"location,omitempty"`
	Items    []XurItem `json:"items"`
}

// XurItem is a single item in Xûr's inventory.
type XurItem struct {
	Hash      string `json:"hash"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Icon      string `json:"icon"`
	Rarity    string `json:"rarity"`
	Missing   bool   `json:"missing"`
	Cost      string `json:"cost"`
	ClassName string `json:"className,omitempty"`
}

// Milestone is one active weekly milestone.
// Missing is nil until per-milestone completion is actually computed (B9) —
// the frontend hides the badge when absent rather than implying "complete".
type Milestone struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Name    string `json:"name"`
	Reward  string `json:"reward"`
	Missing *int   `json:"missing,omitempty"`
	Note    string `json:"note"`
}

// RecommendedAction is one suggested action for the player this week.
type RecommendedAction struct {
	ID     string `json:"id"`
	Text   string `json:"text"`
	Detail string `json:"detail"`
	Badge  string `json:"badge"`
	Done   bool   `json:"done"`
	Diff   string `json:"diff"`
	Time   string `json:"time"`
}

// DailyAction is one actionable item for the "Do This Today" dashboard panel.
type DailyAction struct {
	ID       string   `json:"id"`
	Category string   `json:"category"` // "milestone" | "xur" | "vendor" | "activity"
	Icon     string   `json:"icon"`
	Text     string   `json:"text"`
	Detail   string   `json:"detail"`
	Badge    string   `json:"badge"` // maps to Badge kind on frontend
	ResetsIn Duration `json:"resetsIn"`
	Done     bool     `json:"done"`
}

// --- internal cache types ---

type publicWeeklyCache struct {
	XurItems         []xurItemEnriched
	XurPresent       bool
	XurLeavesAt      time.Time
	ResetAt          time.Time
	MilestoneHashes  []uint32
	MilestoneNames   map[uint32]string
	MilestoneRewards map[uint32]string
	DailyMilestones  []dailyMilestoneItem
	WeeklyActivities []weeklyActivityItem

	// FetchedAt is when this payload was assembled from Bungie data (B8).
	FetchedAt time.Time

	// Degraded marks a payload assembled without manifest enrichment (manifest
	// still downloading). Degraded payloads are cached briefly so the next
	// request after the download self-heals instead of serving fallback labels
	// until the weekly reset.
	Degraded bool
}

type xurItemEnriched struct {
	Hash      uint32
	Name      string
	Type      string
	Icon      string
	Rarity    string
	Cost      string
	ClassType *int
}

type dailyMilestoneItem struct {
	Hash uint32
	Name string
	Desc string
}

type weeklyActivityItem struct {
	Hash      uint32
	Name      string
	Modifiers []string // display names of active modifiers (max 3)
	Category  string   // "Nightfall", "Featured Raid", "Featured Dungeon", "Strike", etc.
}

type dailyVendorItem struct {
	VendorName string
	VendorIcon string // icon key for frontend
	ItemName   string
	ItemHash   uint32
	ItemType   string // "Armor Mod", "Weapon Mod", "Item"
}

// Service assembles the weekly recommendations.
type Service struct {
	bungie      *bungie.Client
	manifest    ManifestRepo
	collections *collections.Service
	wishlist    WishlistReader
	cache       cache.Cache
	efficiency  *efficiency.Engine
	version     versioner
	now         func() time.Time
}

// versioner reports the current manifest version. Satisfied by
// *bungie.ManifestService. Mirrors the identical interface in services/efficiency.
type versioner interface {
	Version() string
}

// manifestVersion returns the installed manifest version, or "none" before the
// first manifest lands. Used to scope cache keys whose values carry
// manifest-resolved labels, so a swap orphans them rather than requiring an
// eviction sweep the cache has no API for.
func (s *Service) manifestVersion() string {
	if s.version == nil {
		return "none"
	}
	if v := s.version.Version(); v != "" {
		return v
	}
	return "none"
}

// ManifestRepo is the subset of the manifest repository the weekly service uses.
// Satisfied by *manifest.Provider; calls error while the manifest is not ready,
// which callers treat the same as a nil repo (fallback labels, no enrichment).
type ManifestRepo interface {
	GetItemsByHashes(hashes []uint32) (map[uint32]*bungie.InventoryItemDefinition, error)
	ResolveVendorLocation(vendorHash uint32, locationIndex int) (uint32, string, error)
	GetMilestoneDefinitions(hashes []uint32) (map[uint32]*bungie.MilestoneDefinition, error)
	GetActivityDefinitions(hashes []uint32) (map[uint32]*bungie.ActivityDefinition, error)
	GetActivityModifierDefinitions(hashes []uint32) (map[uint32]*bungie.ActivityModifierDefinition, error)
}

// WishlistReader is satisfied by *db.WishlistStore.
type WishlistReader interface {
	GetUserID(ctx context.Context, membershipID string) (int64, error)
	List(ctx context.Context, userID int64) ([]WishlistItem, error)
}

// WishlistItem mirrors db.WishlistItem (subset we need).
type WishlistItem struct {
	ItemHash uint32
}

// NewService creates a new weekly recommendations service.
func NewService(b *bungie.Client, m ManifestRepo, c *collections.Service, w WishlistReader, appCache cache.Cache, eng *efficiency.Engine, v versioner) *Service {
	return NewServiceWithClock(b, m, c, w, appCache, eng, v, time.Now)
}

// NewServiceWithClock creates a weekly service with an injected clock. The
// production constructor above always uses time.Now; the alternate constructor
// exists so hermetic browser tests can keep Xur in a deterministic weekend
// window without changing production behavior.
func NewServiceWithClock(b *bungie.Client, m ManifestRepo, c *collections.Service, w WishlistReader, appCache cache.Cache, eng *efficiency.Engine, v versioner, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{
		bungie:      b,
		manifest:    m,
		collections: c,
		wishlist:    w,
		cache:       appCache,
		efficiency:  eng,
		version:     v,
		now:         now,
	}
}

// OnVersionChanged drops the global weekly payload, whose milestone names and
// reward labels are manifest-resolved. Implements bungie.ManifestObserver.
//
// The per-character `daily:vendors:*` entries need no eviction: their key is
// scoped by manifest version, so a swap orphans them automatically.
func (s *Service) OnVersionChanged(version string) error {
	s.cache.Delete(publicWeeklyCacheKey)
	return nil
}

func (s *Service) nowUTC() time.Time {
	if s.now == nil {
		return time.Now().UTC()
	}
	return s.now().UTC()
}

// --- Time math ---

// NextWeeklyReset returns the next Tuesday 17:00 UTC.
func NextWeeklyReset(now time.Time) time.Time {
	now = now.UTC()
	daysUntilTuesday := (2 - int(now.Weekday()) + 7) % 7
	if daysUntilTuesday == 0 && now.Hour() >= 17 {
		daysUntilTuesday = 7
	}
	return time.Date(now.Year(), now.Month(), now.Day()+daysUntilTuesday, 17, 0, 0, 0, time.UTC)
}

// NextDailyReset returns the next daily reset at 17:00 UTC.
func NextDailyReset(now time.Time) time.Time {
	now = now.UTC()
	reset := time.Date(now.Year(), now.Month(), now.Day(), 17, 0, 0, 0, time.UTC)
	if !reset.After(now) {
		reset = reset.Add(24 * time.Hour)
	}
	return reset
}

// NextXurArrival returns the next Friday 17:00 UTC.
func NextXurArrival(now time.Time) time.Time {
	now = now.UTC()
	daysUntilFriday := (5 - int(now.Weekday()) + 7) % 7
	if daysUntilFriday == 0 && now.Hour() >= 17 {
		daysUntilFriday = 7
	}
	return time.Date(now.Year(), now.Month(), now.Day()+daysUntilFriday, 17, 0, 0, 0, time.UTC)
}

// XurPresent returns true during Fri 17:00 UTC through Tue 17:00 UTC.
func XurPresent(now time.Time) bool {
	now = now.UTC()
	wd := now.Weekday()
	h := now.Hour()
	m := now.Minute()
	timeVal := h*60 + m
	switch wd {
	case time.Friday:
		return timeVal >= 17*60
	case time.Saturday, time.Sunday, time.Monday:
		return true
	case time.Tuesday:
		return timeVal < 17*60
	default:
		return false
	}
}

func toDuration(d time.Duration) Duration {
	total := max(int(d.Minutes()), 0)
	days := total / (60 * 24)
	total %= 60 * 24
	hours := total / 60
	mins := total % 60
	return Duration{D: days, H: hours, M: mins}
}

// GetWeekly assembles the weekly payload for a user.
func (s *Service) GetWeekly(ctx context.Context, membershipType int, membershipID, bungieToken, requestedCharacterID string) (*Weekly, error) {
	now := s.nowUTC()
	resetAt := NextWeeklyReset(now)
	resetIn := toDuration(resetAt.Sub(now))
	dailyResetAt := NextDailyReset(now)
	dailyResetIn := toDuration(dailyResetAt.Sub(now))

	resetLabel := resetAt.Format("Tue 15:04 UTC")

	pub, err := s.getPublicWeekly(ctx, now)
	if err != nil {
		observability.Logger(ctx).LogAttrs(ctx, slog.LevelWarn, "weekly public data fetch failed; returning partial response",
			observability.Err(err),
		)
		pub = &publicWeeklyCache{}
	}

	// Per-user missing set and daily vendor items are two independent Bungie
	// round-trips on a cache miss — fetch them concurrently to halve added latency.
	var (
		missingHashes map[uint32]struct{}
		dailyVendors  []dailyVendorItem
		xurLocation   string
		characterID   string
	)
	if bungieToken != "" {
		characterID, _ = s.resolveCharacter(ctx, membershipType, membershipID, bungieToken, requestedCharacterID)
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			mh, mErr := s.collections.GetMissingItemHashes(ctx, membershipType, membershipID, bungieToken)
			if mErr != nil {
				observability.Logger(ctx).LogAttrs(ctx, slog.LevelWarn, "weekly missing items fetch failed",
					slog.Int("membership_type", membershipType),
					observability.ID("membership", membershipID),
					observability.Err(mErr),
				)
				mh = map[uint32]struct{}{}
			}
			missingHashes = mh
		}()
		go func() {
			defer wg.Done()
			// Component 402 may contain class-specific inventory, so this path
			// follows the validated active Guardian.
			dailyVendors = s.getDailyVendorItems(ctx, membershipType, membershipID, characterID, bungieToken, now)
			if pub.XurPresent {
				xurLocation = s.getXurLocation(ctx, membershipType, membershipID, characterID, bungieToken)
			}
		}()
		wg.Wait()
	}
	if missingHashes == nil {
		missingHashes = map[uint32]struct{}{}
	}

	// Per-user wishlist hashes
	wishlistHashes := map[uint32]struct{}{}
	if s.wishlist != nil {
		userID, wlErr := s.wishlist.GetUserID(ctx, membershipID)
		if wlErr == nil {
			items, wlErr := s.wishlist.List(ctx, userID)
			if wlErr == nil {
				for _, it := range items {
					wishlistHashes[it.ItemHash] = struct{}{}
				}
			}
		}
	}

	// Assemble Xûr block
	var xurBlock *Xur
	if pub.XurPresent {
		leavesIn := toDuration(pub.XurLeavesAt.Sub(now))
		items := make([]XurItem, len(pub.XurItems))
		for i, xi := range pub.XurItems {
			_, isMissing := missingHashes[xi.Hash]
			className := ""
			if xi.ClassType != nil && *xi.ClassType >= 0 && *xi.ClassType <= 2 {
				className = bungie.GetClassName(*xi.ClassType)
			}
			items[i] = XurItem{
				Hash:      strconv.FormatUint(uint64(xi.Hash), 10),
				Name:      xi.Name,
				Type:      xi.Type,
				Icon:      xi.Icon,
				Rarity:    xi.Rarity,
				Missing:   isMissing,
				Cost:      xi.Cost,
				ClassName: className,
			}
		}
		xurBlock = &Xur{
			Present:  true,
			LeavesIn: leavesIn,
			Location: xurLocation,
			Items:    items,
		}
	}

	// Assemble weekly milestones (for This Week page), with per-raid missing counts.
	milestones := buildMilestones(pub, s.efficiency, missingHashes)

	recommended := s.rankRecommended(ctx, membershipType, membershipID, characterID, bungieToken, pub, missingHashes, wishlistHashes)
	dailyActions := s.buildDailyActions(pub, dailyVendors, missingHashes, wishlistHashes, now, dailyResetIn, resetIn)

	fetchedAt := pub.FetchedAt
	if fetchedAt.IsZero() {
		fetchedAt = now
	}
	return &Weekly{
		ResetLabel:   resetLabel,
		ResetIn:      resetIn,
		DailyResetIn: dailyResetIn,
		ResetAt:      resetAt,
		FetchedAt:    fetchedAt,
		Degraded:     pub.Degraded,
		Xur:          xurBlock,
		Milestones:   milestones,
		Recommended:  recommended,
		DailyActions: dailyActions,
	}, nil
}

// XurItemHashes returns the set of item hashes Xûr currently sells, for
// wishlist availability checks (B6). Empty when Xûr is absent or the weekly
// data is unavailable. Served from the shared weekly cache; the first call
// after a reset fetches public vendors inline.
func (s *Service) XurItemHashes(ctx context.Context) map[uint32]struct{} {
	return s.xurItemHashesAt(ctx, s.nowUTC())
}

func (s *Service) xurItemHashesAt(ctx context.Context, now time.Time) map[uint32]struct{} {
	out := map[uint32]struct{}{}
	if !XurPresent(now) {
		return out
	}
	pub, err := s.getPublicWeekly(ctx, now)
	if err != nil || pub == nil || !pub.XurPresent {
		return out
	}
	for _, it := range pub.XurItems {
		out[it.Hash] = struct{}{}
	}
	return out
}

// buildMilestones turns the cached public milestones into the wire shape, stamping a
// per-milestone Missing count for raid milestones whose source bucket the efficiency
// engine can match (nil otherwise — the frontend hides the badge). eng may be nil
// (legacy fallback / tests).
func buildMilestones(pub *publicWeeklyCache, eng *efficiency.Engine, missing map[uint32]struct{}) []Milestone {
	milestones := make([]Milestone, 0, len(pub.MilestoneHashes))
	for _, hash := range pub.MilestoneHashes {
		name := pub.MilestoneNames[hash]
		if name == "" {
			continue
		}
		m := Milestone{
			ID:     fmt.Sprintf("m-%d", hash),
			Label:  "Weekly",
			Name:   name,
			Reward: pub.MilestoneRewards[hash],
			Note:   "",
		}
		if eng != nil {
			if count, matched := eng.MissingForMilestone(name, missing); matched {
				c := count
				m.Missing = &c
			}
		}
		milestones = append(milestones, m)
	}
	return milestones
}

func (s *Service) buildDailyActions(pub *publicWeeklyCache, vendors []dailyVendorItem, missing, wishlist map[uint32]struct{}, now time.Time, dailyResetIn, weeklyResetIn Duration) []DailyAction {
	var actions []DailyAction

	// 1. Daily milestones — most time-sensitive, reset every day
	for i, m := range pub.DailyMilestones {
		if len(actions) >= 6 {
			break
		}
		actions = append(actions, DailyAction{
			ID:       fmt.Sprintf("daily-m-%d", i),
			Category: "milestone",
			Icon:     "bolt",
			Text:     m.Name,
			Detail:   "Daily challenge",
			Badge:    "new",
			ResetsIn: dailyResetIn,
		})
	}

	// 2. Xûr — missing or wishlisted items (if he's in town)
	if pub.XurPresent {
		xurLeavesIn := toDuration(pub.XurLeavesAt.Sub(now))
		for i, xi := range pub.XurItems {
			if len(actions) >= 6 {
				break
			}
			_, isMissing := missing[xi.Hash]
			_, inWishlist := wishlist[xi.Hash]
			if !isMissing && !inWishlist {
				continue
			}
			badge := "missing"
			label := "missing from your collection"
			if inWishlist && !isMissing {
				badge = "avail-now"
				label = "on your wishlist"
			}
			actions = append(actions, DailyAction{
				ID:       fmt.Sprintf("daily-xur-%d", i),
				Category: "xur",
				Icon:     "bungie",
				Text:     fmt.Sprintf("Xûr is selling %s", xi.Name),
				Detail:   fmt.Sprintf("%s — %s", xi.Type, label),
				Badge:    badge,
				ResetsIn: xurLeavesIn,
			})
		}
	}

	// 3. Daily vendor mods (Ada-1, Banshee-44)
	for i, v := range vendors {
		if len(actions) >= 6 {
			break
		}
		actions = append(actions, DailyAction{
			ID:       fmt.Sprintf("daily-vendor-%d", i),
			Category: "vendor",
			Icon:     "bolt",
			Text:     fmt.Sprintf("%s is selling %s", v.VendorName, v.ItemName),
			Detail:   v.ItemType,
			Badge:    "new",
			ResetsIn: dailyResetIn,
		})
	}

	// 4. Featured weekly activities with modifiers (Nightfall, etc.)
	for i, a := range pub.WeeklyActivities {
		if len(actions) >= 6 {
			break
		}
		detail := a.Category
		if len(a.Modifiers) > 0 {
			detail += " · " + strings.Join(a.Modifiers, ", ")
		}
		actions = append(actions, DailyAction{
			ID:       fmt.Sprintf("daily-act-%d", i),
			Category: "activity",
			Icon:     "collections",
			Text:     a.Name,
			Detail:   detail,
			Badge:    "in-progress",
			ResetsIn: weeklyResetIn,
		})
	}

	return actions
}

// mapEngineActions converts ranked engine actions into the wire RecommendedAction.
// Reuses existing fields (no new wire shape): Detail = the "why", Badge = availability/
// source, Diff = difficulty from the source string.
func (s *Service) mapEngineActions(actions []efficiency.ScoredAction) []RecommendedAction {
	out := make([]RecommendedAction, 0, len(actions))
	for _, a := range actions {
		badge := "Activity"
		if a.Kind == "vendor" {
			badge = "Vendor"
		}
		switch {
		case a.AvailableNow:
			badge = "Available now"
		case a.WishlistCount > 0:
			badge = "Wishlist"
		}
		out = append(out, RecommendedAction{
			ID:     a.ID,
			Text:   a.Text,
			Detail: a.Why,
			Badge:  badge,
			Done:   false,
			Diff:   collections.ClassifyDifficulty(a.SourceString, false),
			Time:   "",
		})
	}
	return out
}

// rankRecommended runs the efficiency engine; falls back to the legacy Xûr-only
// heuristic when the engine is unavailable or has nothing to suggest (cold index,
// private profile, no missing set). Best-effort — never fails the weekly request.
func (s *Service) rankRecommended(ctx context.Context, membershipType int, membershipID, characterID, bungieToken string, pub *publicWeeklyCache, missing, wishlist map[uint32]struct{}) []RecommendedAction {
	if s.efficiency != nil && bungieToken != "" {
		liveVendors := s.liveVendorItemHashesAtCharacter(ctx, membershipType, membershipID, characterID, bungieToken, s.nowUTC())
		activeMilestones := make([]string, 0, len(pub.MilestoneNames))
		for _, name := range pub.MilestoneNames {
			activeMilestones = append(activeMilestones, name)
		}
		if actions := s.efficiency.Rank(missing, wishlist, liveVendors, activeMilestones); len(actions) > 0 {
			return s.mapEngineActions(actions)
		}
	}
	return s.buildRecommended(pub, missing, wishlist)
}

func (s *Service) buildRecommended(pub *publicWeeklyCache, missing, wishlist map[uint32]struct{}) []RecommendedAction {
	var actions []RecommendedAction

	if pub.XurPresent {
		for i, xi := range pub.XurItems {
			if len(actions) >= 5 {
				break
			}
			if _, inWishlist := wishlist[xi.Hash]; inWishlist {
				actions = append(actions, RecommendedAction{
					ID:     fmt.Sprintf("r-wl-%d", i),
					Text:   fmt.Sprintf("Buy %s from Xûr", xi.Name),
					Detail: fmt.Sprintf("%s — on your wishlist and available now", xi.Type),
					Badge:  "Wishlist",
					Done:   false,
					Diff:   "easy",
					Time:   "5 min",
				})
			} else if _, isMissing := missing[xi.Hash]; isMissing {
				actions = append(actions, RecommendedAction{
					ID:     fmt.Sprintf("r-xur-%d", i),
					Text:   fmt.Sprintf("Buy %s from Xûr", xi.Name),
					Detail: fmt.Sprintf("%s — missing from your collection", xi.Type),
					Badge:  "Xur",
					Done:   false,
					Diff:   "easy",
					Time:   "5 min",
				})
			}
		}
	}

	if len(actions) == 0 {
		actions = append(actions, RecommendedAction{
			ID:     "r-milestones",
			Text:   "Complete weekly milestones before reset",
			Detail: "Earn pinnacle gear and XP before Tuesday 17:00 UTC",
			Badge:  "Weekly",
			Done:   false,
			Diff:   "moderate",
			Time:   "2-3 hrs",
		})
	}

	return actions
}

// publicWeeklyCacheKey holds the global weekly payload, whose milestone names
// and reward labels are resolved through the manifest. Evicted by
// OnVersionChanged — otherwise it serves stale labels until the weekly reset.
const publicWeeklyCacheKey = "weekly:public"

// getPublicWeekly builds or retrieves the cached global weekly payload.
func (s *Service) getPublicWeekly(ctx context.Context, now time.Time) (*publicWeeklyCache, error) {
	const cacheKey = publicWeeklyCacheKey
	if cached, ok := s.cache.Get(cacheKey); ok {
		if p, ok := cached.(*publicWeeklyCache); ok {
			return p, nil
		}
	}

	pub := &publicWeeklyCache{
		XurPresent: XurPresent(now),
		ResetAt:    NextWeeklyReset(now),
		FetchedAt:  now,
	}
	if pub.XurPresent {
		pub.XurLeavesAt = NextWeeklyReset(now)
	} else {
		pub.XurLeavesAt = NextXurArrival(now)
	}

	if pub.XurPresent {
		s.fetchXurInventory(ctx, pub)
	}

	s.fetchMilestones(ctx, pub)

	// TTL = soonest of: weekly reset, Xûr event, daily reset
	ttl := NextWeeklyReset(now).Sub(now)
	if arr := NextXurArrival(now); arr.Sub(now) < ttl {
		ttl = arr.Sub(now)
	}
	if daily := NextDailyReset(now).Sub(now); daily < ttl {
		ttl = daily
	}
	if ttl < 5*time.Minute {
		ttl = 5 * time.Minute
	}
	// A payload built without the manifest must expire quickly so it self-heals
	// once the manifest download finishes (B4).
	if pub.Degraded && ttl > 5*time.Minute {
		ttl = 5 * time.Minute
	}
	s.cache.Set(cacheKey, pub, ttl)
	return pub, nil
}

func (s *Service) fetchXurInventory(ctx context.Context, pub *publicWeeklyCache) {
	vendors, err := s.bungie.GetPublicVendors(ctx)
	if err != nil {
		observability.Logger(ctx).LogAttrs(ctx, slog.LevelWarn, "weekly public vendors fetch failed",
			observability.Err(err),
		)
		return
	}
	xurKey := strconv.FormatUint(uint64(bungie.XurVendorHash), 10)
	xurSales, ok := vendors.Response.Sales.Data[xurKey]
	if !ok {
		observability.Logger(ctx).LogAttrs(ctx, slog.LevelWarn, "Xur not found in public vendors response",
			slog.Uint64("vendor_hash", uint64(bungie.XurVendorHash)),
		)
		return
	}

	var xurHashes []uint32
	hashToCost := map[uint32]string{}
	for _, saleItem := range xurSales.SaleItems {
		xurHashes = append(xurHashes, saleItem.ItemHash)
		if len(saleItem.Costs) > 0 {
			hashToCost[saleItem.ItemHash] = fmt.Sprintf("%d Strange Coin", saleItem.Costs[0].Quantity)
		}
	}

	var defs map[uint32]*bungie.InventoryItemDefinition
	if s.manifest != nil {
		d, err := s.manifest.GetItemsByHashes(xurHashes)
		if err != nil {
			observability.Logger(ctx).LogAttrs(ctx, slog.LevelWarn, "weekly Xur manifest item lookup failed",
				observability.Err(err),
			)
		} else {
			defs = d
		}
	}

	var enriched []xurItemEnriched
	if defs != nil {
		for _, hash := range xurHashes {
			def, ok := defs[hash]
			if !ok || def == nil || def.DisplayProperties.Name == "" {
				continue
			}
			rarity := strings.ToLower(bungie.GetTierName(def.Inventory.TierType))
			enriched = append(enriched, xurItemEnriched{
				Hash:      hash,
				Name:      def.DisplayProperties.Name,
				Type:      bungie.ItemTypeName(def.ItemType, def.ItemSubType),
				Icon:      def.DisplayProperties.Icon,
				Rarity:    rarity,
				Cost:      hashToCost[hash],
				ClassType: armorClassType(def),
			})
		}
	} else {
		pub.Degraded = true
		for _, hash := range xurHashes {
			enriched = append(enriched, xurItemEnriched{Hash: hash, Name: "Unknown Item", Cost: hashToCost[hash]})
		}
	}
	pub.XurItems = enriched
}

func (s *Service) fetchMilestones(ctx context.Context, pub *publicWeeklyCache) {
	milestones, err := s.bungie.GetPublicMilestones(ctx)
	if err != nil {
		observability.Logger(ctx).LogAttrs(ctx, slog.LevelWarn, "weekly public milestones fetch failed",
			observability.Err(err),
		)
		return
	}

	// Collect all hashes for batch manifest lookup
	var hashes []uint32
	for _, m := range milestones {
		hashes = append(hashes, m.MilestoneHash)
	}

	pub.MilestoneNames = make(map[uint32]string)
	pub.MilestoneRewards = make(map[uint32]string)

	// No manifest (nil in degraded mode, or erroring while it downloads) →
	// fall back to hash labels and mark the payload degraded so it expires fast.
	var defs map[uint32]*bungie.MilestoneDefinition
	if s.manifest != nil {
		d, err := s.manifest.GetMilestoneDefinitions(hashes)
		if err != nil {
			observability.Logger(ctx).LogAttrs(ctx, slog.LevelWarn, "weekly milestone manifest lookup failed",
				observability.Err(err),
			)
		} else {
			defs = d
		}
	}
	if defs == nil {
		pub.Degraded = true
		for _, hash := range hashes {
			pub.MilestoneHashes = append(pub.MilestoneHashes, hash)
			pub.MilestoneNames[hash] = fmt.Sprintf("Milestone %d", hash)
		}
		return
	}

	// Collect all activity hashes from weekly milestones for enrichment
	var activityHashes []uint32
	weeklyMilestoneActivities := map[uint32][]bungie.MilestoneActivity{} // milestoneHash → activities
	for _, m := range milestones {
		def, ok := defs[m.MilestoneHash]
		if !ok || def == nil || def.DisplayProperties.Name == "" {
			continue
		}
		if def.MilestoneType == bungie.MilestoneTypeWeekly || def.MilestoneType == bungie.MilestoneTypeSpecial {
			for _, a := range m.Activities {
				activityHashes = append(activityHashes, a.ActivityHash)
			}
			if len(m.Activities) > 0 {
				weeklyMilestoneActivities[m.MilestoneHash] = m.Activities
			}
		}
	}

	// Batch-fetch activity definitions and modifiers
	activityDefs := map[uint32]*bungie.ActivityDefinition{}
	modifierDefs := map[uint32]*bungie.ActivityModifierDefinition{}
	if len(activityHashes) > 0 {
		if ad, err := s.manifest.GetActivityDefinitions(activityHashes); err == nil {
			activityDefs = ad
		}
		// Collect all modifier hashes across all activities
		var modHashes []uint32
		for _, m := range milestones {
			for _, a := range m.Activities {
				modHashes = append(modHashes, a.ModifierHashes...)
			}
		}
		if len(modHashes) > 0 {
			if md, err := s.manifest.GetActivityModifierDefinitions(modHashes); err == nil {
				modifierDefs = md
			}
		}
	}

	// Populate pub caches
	for _, m := range milestones {
		def, ok := defs[m.MilestoneHash]
		if !ok || def == nil || def.DisplayProperties.Name == "" {
			continue
		}
		name := def.DisplayProperties.Name

		switch def.MilestoneType {
		case bungie.MilestoneTypeDaily:
			pub.DailyMilestones = append(pub.DailyMilestones, dailyMilestoneItem{
				Hash: m.MilestoneHash,
				Name: name,
				Desc: def.DisplayProperties.Description,
			})

		case bungie.MilestoneTypeWeekly, bungie.MilestoneTypeSpecial:
			pub.MilestoneHashes = append(pub.MilestoneHashes, m.MilestoneHash)
			pub.MilestoneNames[m.MilestoneHash] = name
			pub.MilestoneRewards[m.MilestoneHash] = "Pinnacle Gear"

			// Build weekly activity entries (with modifiers) for Do This Today
			acts, hasActs := weeklyMilestoneActivities[m.MilestoneHash]
			if !hasActs {
				continue
			}
			for _, a := range acts {
				actDef, ok := activityDefs[a.ActivityHash]
				if !ok || actDef == nil {
					continue
				}
				actName := actDef.DisplayProperties.Name
				if actName == "" {
					actName = name
				}
				// Only surface activities that have modifiers — those are the ones
				// that communicate meaningful loadout information to the player.
				if len(a.ModifierHashes) == 0 {
					continue
				}
				var modNames []string
				for _, mh := range a.ModifierHashes {
					md, ok := modifierDefs[mh]
					if !ok || md == nil || md.DisplayProperties.Name == "" {
						continue
					}
					modNames = append(modNames, md.DisplayProperties.Name)
					if len(modNames) == 3 {
						break
					}
				}
				pub.WeeklyActivities = append(pub.WeeklyActivities, weeklyActivityItem{
					Hash:      a.ActivityHash,
					Name:      actName,
					Modifiers: modNames,
					Category:  categoryFromMilestoneName(name),
				})
			}
		}
	}
}

// dailyVendorsCacheKey scopes the cached daily-vendor rows by manifest version,
// because those rows carry manifest-resolved item names and types. A swap
// therefore orphans the old entries (they expire on their existing TTL) rather
// than needing an eviction sweep the cache has no API for, and the rebuild is
// cheap: the underlying Bungie vendor response is cached separately under an
// unversioned key, so nothing refetches from Bungie.
//
// A method rather than a bare fmt.Sprintf at the use site so tests seed the
// cache through the same constructor instead of transcribing the format.
func (s *Service) dailyVendorsCacheKey(membershipType int, membershipID, characterID string) string {
	return fmt.Sprintf("daily:vendors:%s:%d:%s:%s", s.manifestVersion(), membershipType, membershipID, characterID)
}

// getDailyVendorItems fetches Ada-1 and Banshee-44 inventory via the character vendor endpoint.
// Component 402 can vary by class, so the result is cached per validated character.
func (s *Service) getDailyVendorItems(ctx context.Context, membershipType int, membershipID, characterID, bungieToken string, now time.Time) []dailyVendorItem {
	cacheKey := s.dailyVendorsCacheKey(membershipType, membershipID, characterID)
	if cached, ok := s.cache.Get(cacheKey); ok {
		if items, ok := cached.([]dailyVendorItem); ok {
			return items
		}
	}

	resp := s.getCharacterVendors(ctx, membershipType, membershipID, characterID, bungieToken)
	if resp == nil {
		return nil
	}

	items := s.enrichDailyVendorItems(resp)

	// Only cache non-empty results — a transient empty response would poison the cache
	// for all users until the next daily reset.
	if len(items) > 0 {
		ttl := max(NextDailyReset(now).Sub(now), 5*time.Minute)
		s.cache.Set(cacheKey, items, ttl)
	}
	return items
}

const xurTowerDestinationHash uint32 = 1737926756

// getXurLocation resolves Xûr's character-scoped live location through the
// manifest. Location is best-effort: failures return an empty string so the
// weekly payload can omit the field while preserving inventory and schedule.
func (s *Service) getXurLocation(ctx context.Context, membershipType int, membershipID, characterID, bungieToken string) string {
	resp := s.getCharacterVendors(ctx, membershipType, membershipID, characterID, bungieToken)
	if resp == nil || s.manifest == nil {
		return ""
	}

	xurKey := strconv.FormatUint(uint64(bungie.XurVendorHash), 10)
	vendor, ok := resp.Response.Vendors.Data[xurKey]
	if !ok || !vendor.Enabled || vendor.VendorLocationIndex < 0 {
		return ""
	}

	destinationHash, destinationName, err := s.manifest.ResolveVendorLocation(
		bungie.XurVendorHash,
		vendor.VendorLocationIndex,
	)
	if err != nil {
		observability.Logger(ctx).LogAttrs(ctx, slog.LevelWarn, "weekly Xur location lookup failed",
			observability.Err(err),
		)
		return ""
	}
	return xurLocationLabel(destinationHash, destinationName)
}

func xurLocationLabel(destinationHash uint32, destinationName string) string {
	if destinationHash == xurTowerDestinationHash {
		return "The Tower"
	}
	return destinationName
}

type characterRoster struct {
	Characters map[string]bungie.CharacterComponent
	PrimaryID  string
}

// resolveCharacter validates a requested character against the authenticated
// roster and fails soft to the most recently played character on mismatch.
func (s *Service) resolveCharacter(ctx context.Context, membershipType int, membershipID, bungieToken, requestedCharacterID string) (string, int) {
	cacheKey := fmt.Sprintf("weekly:roster:%d:%s", membershipType, membershipID)
	if cached, ok := s.cache.Get(cacheKey); ok {
		if roster, ok := cached.(characterRoster); ok {
			return roster.resolve(requestedCharacterID)
		}
	}
	if s.bungie == nil {
		return "", -1
	}

	chars, err := s.bungie.GetCharacters(ctx, membershipType, membershipID, bungieToken)
	if err != nil {
		observability.Logger(ctx).LogAttrs(ctx, slog.LevelWarn, "weekly character roster fetch failed",
			slog.Int("membership_type", membershipType),
			observability.ID("membership", membershipID),
			observability.Err(err),
		)
		return "", -1
	}

	roster := characterRoster{Characters: make(map[string]bungie.CharacterComponent)}
	var mostRecentTime time.Time
	for id, c := range chars.Response.Characters.Data {
		if c.CharacterID == "" {
			c.CharacterID = id
		}
		roster.Characters[id] = c
		t, err := time.Parse(time.RFC3339, c.DateLastPlayed)
		if roster.PrimaryID == "" || (err == nil && t.After(mostRecentTime)) {
			mostRecentTime = t
			roster.PrimaryID = id
		}
	}

	if len(roster.Characters) > 0 {
		s.cache.Set(cacheKey, roster, 5*time.Minute)
	}
	return roster.resolve(requestedCharacterID)
}

func (r characterRoster) resolve(requestedCharacterID string) (string, int) {
	if c, ok := r.Characters[requestedCharacterID]; requestedCharacterID != "" && ok {
		return requestedCharacterID, c.ClassType
	}
	c, ok := r.Characters[r.PrimaryID]
	if !ok {
		return "", -1
	}
	return r.PrimaryID, c.ClassType
}

func armorClassType(def *bungie.InventoryItemDefinition) *int {
	if def == nil || def.ItemType != bungie.ItemTypeArmor || def.ClassType == nil || *def.ClassType < 0 || *def.ClassType > 2 {
		return nil
	}
	classType := *def.ClassType
	return &classType
}

func (s *Service) enrichDailyVendorItems(resp *bungie.CharacterVendorsResponse) []dailyVendorItem {
	type vendorMeta struct {
		name     string
		icon     string
		itemType string // label for the kind of thing they sell
	}
	vendorMetas := map[string]vendorMeta{
		strconv.FormatUint(uint64(bungie.Ada1VendorHash), 10):      {name: "Ada-1", icon: "bolt", itemType: "Armor Mod"},
		strconv.FormatUint(uint64(bungie.Banshee44VendorHash), 10): {name: "Banshee-44", icon: "bolt", itemType: "Weapon Mod"},
	}

	// Collect all item hashes from the two vendors
	type pending struct {
		vendorKey string
		hash      uint32
	}
	var pendings []pending
	for vendorKey, meta := range vendorMetas {
		sales, ok := resp.Response.Sales.Data[vendorKey]
		if !ok {
			slog.Warn("weekly vendor not found in character vendors response",
				slog.String("vendor", meta.name),
			)
			continue
		}
		for _, sale := range sales.SaleItems {
			if sale.ItemHash == 0 {
				continue
			}
			pendings = append(pendings, pending{vendorKey: vendorKey, hash: sale.ItemHash})
		}
	}
	if len(pendings) == 0 {
		return nil
	}

	// Batch-fetch item definitions
	allHashes := make([]uint32, len(pendings))
	for i, p := range pendings {
		allHashes[i] = p.hash
	}
	var itemDefs map[uint32]*bungie.InventoryItemDefinition
	if s.manifest != nil {
		itemDefs, _ = s.manifest.GetItemsByHashes(allHashes)
	}

	var items []dailyVendorItem
	for _, p := range pendings {
		def, hasDef := itemDefs[p.hash]
		// Only surface actual mods — vendors sell enhancement cores, upgrade modules, etc.
		// that are not actionable in the "Do This Today" context. Items with no
		// manifest definition are dropped too (stale manifest → junk "Unknown Mod" rows).
		if !hasDef || def == nil || def.ItemType != bungie.ItemTypeMod {
			continue
		}
		meta := vendorMetas[p.vendorKey]
		name := def.DisplayProperties.Name
		if name == "" {
			continue
		}
		items = append(items, dailyVendorItem{
			VendorName: meta.name,
			VendorIcon: meta.icon,
			ItemName:   name,
			ItemHash:   p.hash,
			ItemType:   meta.itemType,
		})
	}
	return items
}

func categoryFromMilestoneName(name string) string {
	return sources.MilestoneCategory(name)
}
