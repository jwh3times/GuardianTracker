package db

import "github.com/jackc/pgx/v5/pgxpool"

// Stores holds all DB store instances. Fields are nil when pool is nil (degraded mode).
type Stores struct {
	Users    *UserStore
	Tokens   *BungieTokenStore
	Wishlist *WishlistStore
	Prefs    *PrefsStore
}

func NewStores(pool *pgxpool.Pool) *Stores {
	if pool == nil {
		return &Stores{}
	}
	return &Stores{
		Users:    NewUserStore(pool),
		Tokens:   NewBungieTokenStore(pool),
		Wishlist: NewWishlistStore(pool),
		Prefs:    NewPrefsStore(pool),
	}
}
