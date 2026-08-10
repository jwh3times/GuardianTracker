package db

import "github.com/jackc/pgx/v5/pgxpool"

// Stores holds every persistence seam the service uses.
//
// The fields are interfaces, never nil. Without a database (no DATABASE_URL)
// they hold degraded implementations whose every method returns ErrUnavailable,
// so a caller cannot forget to check: there is no nil to dereference and no
// second code path to remember. Ask Available() when you genuinely need to know
// whether persistence exists — starting a pruner, or deciding whether role
// claims are authoritative — rather than nil-testing a field.
type Stores struct {
	Users    UserRepo
	Tokens   TokenRepo
	Wishlist WishlistRepo
	Prefs    PrefsRepo
	Flags    FlagRepo
	Audit    AuditRepo
	Pinger   Pinger

	available bool
}

// Available reports whether the stores are backed by a real database.
func (s *Stores) Available() bool { return s != nil && s.available }

func NewStores(pool *pgxpool.Pool) *Stores {
	if pool == nil {
		return &Stores{
			Users:    degradedUsers{},
			Tokens:   degradedTokens{},
			Wishlist: degradedWishlist{},
			Prefs:    degradedPrefs{},
			Flags:    degradedFlags{},
			Audit:    degradedAudit{},
			Pinger:   degradedPinger{},
		}
	}
	return &Stores{
		Users:     NewUserStore(pool),
		Tokens:    NewBungieTokenStore(pool),
		Wishlist:  NewWishlistStore(pool),
		Prefs:     NewPrefsStore(pool),
		Flags:     NewFlagStore(pool),
		Audit:     NewAuditStore(pool),
		Pinger:    pool,
		available: true,
	}
}
