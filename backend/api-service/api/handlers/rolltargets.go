package handlers

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"

	"guardian-tracker/api-service/observability"
	"guardian-tracker/api-service/services/bungie"
	"guardian-tracker/api-service/services/ownedrolls"
	"guardian-tracker/api-service/services/rolltargets"

	"github.com/gin-gonic/gin"
)

// maxImportBytes bounds one DIM-format upload. A community wish list covering
// most of the game runs to a few hundred kilobytes; this leaves generous room
// and still refuses a file that was never a wish list. The router's global body
// limit applies too — this is the narrower, clearer refusal.
const maxImportBytes = 4 << 20 // 4 MiB

// rollTargetService is the roll-target capability this handler drives.
// Satisfied by *rolltargets.Service, which owns validation, the weapon perk
// pool check, persistence, and DIM parsing — the handler owns none of them.
type rollTargetService interface {
	List(ctx context.Context, membershipID string) ([]rolltargets.StoredTarget, error)
	Add(ctx context.Context, membershipID string, cmd rolltargets.AddCommand) (rolltargets.StoredTarget, error)
	Update(ctx context.Context, membershipID string, id rolltargets.TargetID, patch rolltargets.UpdateCommand) (rolltargets.StoredTarget, error)
	Remove(ctx context.Context, membershipID string, id rolltargets.TargetID) error
	ImportDIM(ctx context.Context, membershipID, text string) (rolltargets.ImportReport, error)
	Matches(ctx context.Context, membershipType int, membershipID string) (rolltargets.MatchReport, error)
}

// RollTargetsHandler adapts roll targets to HTTP. It owns request binding,
// error-to-status mapping and serialization, and nothing else.
type RollTargetsHandler struct {
	targets rollTargetService
}

func NewRollTargetsHandler(svc rollTargetService) *RollTargetsHandler {
	return &RollTargetsHandler{targets: svc}
}

// membershipIDOf reads the caller's Destiny membership from the JWT the
// middleware validated. Roll targets are never addressed by client-supplied
// identity: there is no membership on the route or in any request body.
func membershipIDOf(c *gin.Context) string { return c.GetString("membership_id") }

// rollTargetResponse is the JSON shape returned to clients.
//
// ItemHash is a pointer so an any-weapon target serialises as an explicit null
// rather than as item 0, which is a hash the wire would accept.
type rollTargetResponse struct {
	ID        string   `json:"id"`
	ItemHash  *uint32  `json:"itemHash"`
	AnyWeapon bool     `json:"anyWeapon"`
	Wanted    bool     `json:"wanted"`
	Perks     []string `json:"perks"`
	Notes     string   `json:"notes"`
	DateAdded string   `json:"dateAdded"`
}

func rollTargetResponses(targets []rolltargets.StoredTarget) []rollTargetResponse {
	out := make([]rollTargetResponse, 0, len(targets))
	for _, t := range targets {
		out = append(out, rollTargetResponseOf(t))
	}
	return out
}

func rollTargetResponseOf(t rolltargets.StoredTarget) rollTargetResponse {
	perks := t.Perks
	if perks == nil {
		perks = []string{} // serialize as [] not null
	}
	return rollTargetResponse{
		ID:        strconv.FormatInt(int64(t.ID), 10),
		ItemHash:  t.ItemHash,
		AnyWeapon: t.AnyWeapon(),
		Wanted:    t.Wanted,
		Perks:     perks,
		Notes:     t.Notes,
		DateAdded: t.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}

// GetRollTargets handles GET /api/rolltargets
func (h *RollTargetsHandler) GetRollTargets(c *gin.Context) {
	targets, err := h.targets.List(c.Request.Context(), membershipIDOf(c))
	if err != nil {
		handleRollTargetError(c, err, "roll target listing failed")
		return
	}
	c.JSON(http.StatusOK, rollTargetResponses(targets))
}

// addRollTargetRequest is the create body.
//
// ItemHash is a pointer because omitting it is meaningful: it saves an
// any-weapon target. Wanted is a pointer for the same reason — a missing field
// must take the default of true rather than read as false, which would silently
// invert what the caller asked for.
type addRollTargetRequest struct {
	ItemHash *uint32  `json:"itemHash"`
	Wanted   *bool    `json:"wanted"`
	Perks    []string `json:"perks"`
	Notes    string   `json:"notes"`
}

// AddRollTarget handles POST /api/rolltargets
func (h *RollTargetsHandler) AddRollTarget(c *gin.Context) {
	var req addRollTargetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	wanted := true
	if req.Wanted != nil {
		wanted = *req.Wanted
	}
	target, err := h.targets.Add(c.Request.Context(), membershipIDOf(c), rolltargets.AddCommand{
		ItemHash: req.ItemHash,
		Wanted:   wanted,
		Perks:    req.Perks,
		Notes:    req.Notes,
	})
	if err != nil {
		handleRollTargetError(c, err, "roll target create failed")
		return
	}
	c.JSON(http.StatusCreated, rollTargetResponseOf(target))
}

// updateRollTargetRequest is the patch body: a nil field is unchanged, which is
// what distinguishes "leave the notes alone" from "clear the notes".
type updateRollTargetRequest struct {
	Perks *[]string `json:"perks"`
	Notes *string   `json:"notes"`
}

// UpdateRollTarget handles PATCH /api/rolltargets/:id
func (h *RollTargetsHandler) UpdateRollTarget(c *gin.Context) {
	id, ok := rollTargetIDParam(c)
	if !ok {
		return
	}
	var req updateRollTargetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	target, err := h.targets.Update(c.Request.Context(), membershipIDOf(c), id, rolltargets.UpdateCommand{
		Perks: req.Perks,
		Notes: req.Notes,
	})
	if err != nil {
		handleRollTargetError(c, err, "roll target update failed")
		return
	}
	c.JSON(http.StatusOK, rollTargetResponseOf(target))
}

// RemoveRollTarget handles DELETE /api/rolltargets/:id
func (h *RollTargetsHandler) RemoveRollTarget(c *gin.Context) {
	id, ok := rollTargetIDParam(c)
	if !ok {
		return
	}
	if err := h.targets.Remove(c.Request.Context(), membershipIDOf(c), id); err != nil {
		handleRollTargetError(c, err, "roll target delete failed")
		return
	}
	c.Status(http.StatusNoContent)
}

// importLineResponse is one line's fate on the wire.
type importLineResponse struct {
	Line     int      `json:"line"`
	Outcome  string   `json:"outcome"`
	Detail   string   `json:"detail,omitempty"`
	ItemHash *uint32  `json:"itemHash,omitempty"`
	Wanted   bool     `json:"wanted"`
	Perks    []string `json:"perks,omitempty"`
}

// importReportResponse carries every line, plus a tally for a caller that only
// wants the headline. The per-line list is not optional: a line that did not
// import has to be visible, or it is indistinguishable from one that did.
type importReportResponse struct {
	Title       string               `json:"title,omitempty"`
	Description string               `json:"description,omitempty"`
	Imported    int                  `json:"imported"`
	Counts      map[string]int       `json:"counts"`
	Lines       []importLineResponse `json:"lines"`
}

// ImportRollTargets handles POST /api/rolltargets/import
//
// The body is the DIM-format text itself rather than a JSON wrapper: it is a
// file the owner exported, and re-encoding it to paste into a JSON string only
// creates a way to corrupt it.
func (h *RollTargetsHandler) ImportRollTargets(c *gin.Context) {
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, maxImportBytes+1))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "could not read request body"})
		return
	}
	if len(body) > maxImportBytes {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "wish list file is too large"})
		return
	}
	if len(body) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "wish list file is empty"})
		return
	}

	report, err := h.targets.ImportDIM(c.Request.Context(), membershipIDOf(c), string(body))
	if err != nil {
		handleRollTargetError(c, err, "roll target import failed")
		return
	}
	c.JSON(http.StatusOK, importResponseOf(report))
}

func importResponseOf(r rolltargets.ImportReport) importReportResponse {
	counts := map[string]int{}
	for outcome, n := range r.Counts() {
		counts[string(outcome)] = n
	}
	lines := make([]importLineResponse, 0, len(r.Lines))
	for _, l := range r.Lines {
		lines = append(lines, importLineResponse{
			Line:     l.Number,
			Outcome:  string(l.Outcome),
			Detail:   l.Detail,
			ItemHash: l.ItemHash,
			Wanted:   l.Wanted,
			Perks:    l.Perks,
		})
	}
	return importReportResponse{
		Title:       r.Title,
		Description: r.Description,
		Imported:    r.Imported(),
		Counts:      counts,
		Lines:       lines,
	}
}

// matchResponse is one owned weapon satisfying one saved roll.
type matchResponse struct {
	TargetID   string   `json:"targetId"`
	ItemHash   uint32   `json:"itemHash"`
	InstanceID string   `json:"instanceId"`
	Perks      []string `json:"perks"`
	TargetName []string `json:"targetPerks"`
	Notes      string   `json:"notes,omitempty"`
}

// matchReportResponse answers "which of my weapons match what I saved".
//
// unmatchedTargets is not an afterthought: a saved roll that nothing satisfies
// is the roll still worth chasing, and a response carrying only matches would
// hide the most useful thing the feature has to say.
type matchReportResponse struct {
	Wanted           []matchResponse      `json:"wanted"`
	Unwanted         []matchResponse      `json:"unwanted"`
	UnmatchedTargets []rollTargetResponse `json:"unmatchedTargets"`
}

// GetRollTargetMatches handles GET /api/rolltargets/matches
func (h *RollTargetsHandler) GetRollTargetMatches(c *gin.Context) {
	report, err := h.targets.Matches(c.Request.Context(), c.GetInt("membership_type"), membershipIDOf(c))
	if err != nil {
		handleRollTargetError(c, err, "roll target matching failed")
		return
	}
	c.JSON(http.StatusOK, matchReportResponse{
		Wanted:           matchResponses(report.Wanted),
		Unwanted:         matchResponses(report.Unwanted),
		UnmatchedTargets: rollTargetResponses(report.UnmatchedTargets),
	})
}

func matchResponses(matches []rolltargets.Match) []matchResponse {
	out := make([]matchResponse, 0, len(matches))
	for _, m := range matches {
		out = append(out, matchResponse{
			TargetID:   strconv.FormatInt(int64(m.Target.ID), 10),
			ItemHash:   m.Roll.ItemHash,
			InstanceID: m.Roll.InstanceID,
			Perks:      m.Roll.Perks,
			TargetName: m.Target.Perks,
			Notes:      m.Target.Notes,
		})
	}
	return out
}

// rollTargetIDParam reads and validates the :id path parameter, answering the
// request itself when it is not a target id.
func rollTargetIDParam(c *gin.Context) (rolltargets.TargetID, bool) {
	raw := c.Param("id")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid roll target id"})
		return 0, false
	}
	return rolltargets.TargetID(id), true
}

// handleRollTargetError maps the domain's vocabulary onto status codes.
//
// A target that is missing and one that belongs to another membership are both
// 404 on purpose: the domain already refuses to tell them apart, and a 403 here
// would confirm another user's target ids.
func handleRollTargetError(c *gin.Context, err error, logMsg string) {
	switch {
	case errors.Is(err, rolltargets.ErrUnavailable):
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "This feature needs the user data database, which isn't configured on this server.",
			"code":  "DB_UNAVAILABLE",
		})
	case errors.Is(err, rolltargets.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "roll target not found"})
	case errors.Is(err, rolltargets.ErrDuplicate):
		c.JSON(http.StatusConflict, gin.H{"error": "this roll is already saved"})
	case errors.Is(err, rolltargets.ErrOwnedRollsUnavailable):
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Your Destiny inventory could not be read, so matches cannot be shown.",
			"code":  "OWNED_ROLLS_UNAVAILABLE",
		})
	case errors.Is(err, ownedrolls.ErrNoCredential):
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Reconnect your Bungie account to read what you own.",
			"code":  "BUNGIE_REAUTH_REQUIRED",
		})
	case errors.Is(err, ownedrolls.ErrInventoryUnavailable):
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Your Destiny inventory could not be read, so matches cannot be shown.",
			"code":  "OWNED_ROLLS_UNAVAILABLE",
		})
	case errors.Is(err, rolltargets.ErrPerksUnavailable), errors.Is(err, ownedrolls.ErrPerksUnavailable):
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "The item database is still downloading — try again in a moment.",
			"code":  "MANIFEST_NOT_READY",
		})
	case isRollTargetValidationError(err):
		c.JSON(http.StatusBadRequest, gin.H{"error": rollTargetValidationMessage(err)})
	case isBungieError(err):
		// The owned-roll read's profile request failed upstream. Answer it the
		// way every other Bungie-backed route does, so a rate limit reads as a
		// rate limit rather than as a fault in this server.
		handleBungieError(c, err)
	default:
		ctx := handlerContext(c)
		observability.Logger(ctx).ErrorContext(ctx, logMsg, observability.Err(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error", "code": "INTERNAL_ERROR"})
	}
}

func isBungieError(err error) bool {
	var bungieErr *bungie.BungieError
	return errors.As(err, &bungieErr)
}

func isRollTargetValidationError(err error) bool {
	for _, sentinel := range []error{
		rolltargets.ErrNoPerks,
		rolltargets.ErrTooManyPerks,
		rolltargets.ErrDuplicatePerk,
		rolltargets.ErrNotesTooLong,
		rolltargets.ErrNotAWeapon,
		rolltargets.ErrUnknownPerk,
		rolltargets.ErrUnknownPerkName,
	} {
		if errors.Is(err, sentinel) {
			return true
		}
	}
	return false
}

// rollTargetValidationMessage keeps the refusal readable. The domain errors
// carry a package prefix for logs; the wire never has.
func rollTargetValidationMessage(err error) string {
	switch {
	case errors.Is(err, rolltargets.ErrNoPerks):
		return "at least one perk is required"
	case errors.Is(err, rolltargets.ErrTooManyPerks):
		return "at most 10 perks per roll target"
	case errors.Is(err, rolltargets.ErrDuplicatePerk):
		return "the same perk is named twice"
	case errors.Is(err, rolltargets.ErrNotesTooLong):
		return "notes must be 500 characters or fewer"
	case errors.Is(err, rolltargets.ErrNotAWeapon):
		return "item is not a weapon with perk columns"
	case errors.Is(err, rolltargets.ErrUnknownPerk):
		return "this weapon cannot roll one of the named perks"
	case errors.Is(err, rolltargets.ErrUnknownPerkName):
		return "no perk has that name"
	}
	return "invalid request"
}
