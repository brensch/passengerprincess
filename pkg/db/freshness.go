package db

import "time"

const (
	// MaxContentAge is how long Places content may be served after it was fetched.
	// The Google Maps Platform terms limit caching of place content to 30 days.
	MaxContentAge = 30 * 24 * time.Hour

	// MaxVerifyAge is how long a supercharger and its restaurants may be served after they were
	// last confirmed against the Places API.
	MaxVerifyAge = 72 * time.Hour

	// MissedChecksBeforeRemoval is how many consecutive incomplete restaurant searches must omit a
	// restaurant before it is unlinked from a supercharger.
	MissedChecksBeforeRemoval = 2
)
