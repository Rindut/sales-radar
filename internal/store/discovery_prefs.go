package store

import (
	"database/sql"

	"salesradar/internal/domain"
)

// Keys for discovery_source_prefs.source_key (stable API for settings UI + DB).
const (
	PrefGoogle       = "google"
	PrefSeed         = "seed"
	PrefWebsiteCrawl = "website_crawl"
	PrefJobSignal    = "job_signal"
	PrefApollo       = "apollo"
	PrefLinkedIn     = "linkedin"
)

// GetDiscoverySourceToggles loads saved preferences merged with defaults (missing rows default to enabled).
func GetDiscoverySourceToggles(db *sql.DB) (domain.DiscoverySourceToggles, error) {
	t := domain.DefaultDiscoverySourceToggles()
	rows, err := db.Query(`SELECT source_key, enabled FROM discovery_source_prefs`)
	if err != nil {
		return t, err
	}
	defer rows.Close()
	for rows.Next() {
		var key string
		var en int
		if err := rows.Scan(&key, &en); err != nil {
			return t, err
		}
		on := en != 0
		switch key {
		case PrefGoogle:
			t.Google = on
		case PrefSeed:
			t.Seed = on
		case PrefWebsiteCrawl:
			t.WebsiteCrawl = on
		case PrefJobSignal:
			t.JobSignal = on
		case PrefApollo:
			t.Apollo = on
		case PrefLinkedIn:
			t.LinkedIn = on
		}
	}
	return t, rows.Err()
}

// SetDiscoverySourceToggles persists all six toggles.
func SetDiscoverySourceToggles(db *sql.DB, t domain.DiscoverySourceToggles) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	upsert := func(key string, on bool) error {
		en := 0
		if on {
			en = 1
		}
		_, err := tx.Exec(`
			INSERT INTO discovery_source_prefs (source_key, enabled) VALUES (?, ?)
			ON CONFLICT(source_key) DO UPDATE SET enabled = excluded.enabled`,
			key, en)
		return err
	}
	if err := upsert(PrefGoogle, t.Google); err != nil {
		return err
	}
	if err := upsert(PrefSeed, t.Seed); err != nil {
		return err
	}
	if err := upsert(PrefWebsiteCrawl, t.WebsiteCrawl); err != nil {
		return err
	}
	if err := upsert(PrefJobSignal, t.JobSignal); err != nil {
		return err
	}
	if err := upsert(PrefApollo, t.Apollo); err != nil {
		return err
	}
	if err := upsert(PrefLinkedIn, t.LinkedIn); err != nil {
		return err
	}
	return tx.Commit()
}
