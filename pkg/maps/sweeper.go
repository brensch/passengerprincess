package maps

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/brensch/passengerprincess/pkg/db"
)

const (
	// SweepInterval is how often the sweeper looks for superchargers due a refresh or verification.
	SweepInterval = time.Hour

	// VerifyInterval is how often a supercharger and its restaurants are checked against the Places API.
	// It is well inside db.MaxVerifyAge so a few failed sweeps do not hide anything.
	VerifyInterval = 24 * time.Hour

	// RefreshAge is when a supercharger's details are re-fetched. It leaves a margin before
	// db.MaxContentAge for the sweeper to work through a backlog.
	RefreshAge = 27 * 24 * time.Hour

	// sweepRefreshBatch caps paid refreshes per sweep; sweepVerifyBatch caps free verifications.
	sweepRefreshBatch = 50
	sweepVerifyBatch  = 100

	// sweeperIPAddress identifies sweeper calls in the maps call log.
	sweeperIPAddress = "sweeper"
	sweeperRequestID = "sweeper"
)

// RunSweeper keeps cached superchargers fresh until ctx is cancelled, sweeping immediately and
// then every SweepInterval.
func RunSweeper(ctx context.Context, broker *db.Service, apiKey string) {
	ticker := time.NewTicker(SweepInterval)
	defer ticker.Stop()
	for {
		Sweep(ctx, broker, apiKey, time.Now())
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// Sweep re-fetches superchargers whose details are older than RefreshAge, then verifies those not
// verified within VerifyInterval. Failures are logged and retried on the next sweep.
func Sweep(ctx context.Context, broker *db.Service, apiKey string, now time.Time) {
	refreshed, verified, failed := 0, 0, 0

	due, err := broker.Supercharger.ListUpdatedBefore(now.Add(-RefreshAge), sweepRefreshBatch)
	if err != nil {
		slog.Error("Sweep failed to list superchargers due a refresh", "error", err)
	}
	for _, sc := range due {
		if _, _, err := fetchSupercharger(ctx, broker, apiKey, sc.PlaceID, sweeperIPAddress, sweeperRequestID); err != nil {
			slog.Error("Sweep failed to refresh supercharger", "place_id", sc.PlaceID, "error", err)
			failed++
			continue
		}
		refreshed++
	}

	due, err = broker.Supercharger.ListVerifiedBefore(now.Add(-VerifyInterval), sweepVerifyBatch)
	if err != nil {
		slog.Error("Sweep failed to list superchargers due verification", "error", err)
	}
	for _, sc := range due {
		if err := verifySupercharger(ctx, broker, apiKey, &sc, now); err != nil {
			slog.Error("Sweep failed to verify supercharger", "place_id", sc.PlaceID, "error", err)
			failed++
			continue
		}
		verified++
	}

	slog.Info("Sweep completed", "refreshed", refreshed, "verified", verified, "failed", failed)
}

// verifySupercharger confirms a cached supercharger still exists and reconciles its restaurants,
// using only free IDs Only calls unless the restaurant search has new places.
func verifySupercharger(ctx context.Context, broker *db.Service, apiKey string, supercharger *db.Supercharger, now time.Time) error {
	_, err := GetPlaceDetails(ctx, broker, apiKey, supercharger.PlaceID, FieldMaskPlaceIDOnly, sweeperIPAddress, sweeperRequestID)
	if errors.Is(err, ErrPlaceNotFound) {
		slog.Warn("Supercharger no longer exists", "place_id", supercharger.PlaceID)
		return broker.Supercharger.Delete(supercharger.PlaceID)
	}
	if err != nil {
		return err
	}

	location := Point{Latitude: supercharger.Latitude, Longitude: supercharger.Longitude}
	places, truncated, err := GetPlacesViaTextSearch(ctx, broker, apiKey, restaurantSearchQuery, FieldMaskRestaurantIDsOnly,
		Circle{Center: location, Radius: restaurantWalkingRadiusMeters}, true, sweeperIPAddress, sweeperRequestID)
	if err != nil {
		return err
	}

	known := make(map[string]bool, len(supercharger.RestaurantSearchIDs))
	for _, id := range supercharger.RestaurantSearchIDs {
		known[id] = true
	}
	returnedIDs := make([]string, 0, len(places))
	hasNewPlace := false
	for _, place := range places {
		returnedIDs = append(returnedIDs, place.ID)
		hasNewPlace = hasNewPlace || !known[place.ID]
	}

	// A truncated search can surface different places from one call to the next, so only a
	// complete search is trusted to show a place is new.
	if hasNewPlace && !truncated {
		_, err := refreshRestaurants(ctx, broker, apiKey, supercharger, now, sweeperIPAddress, sweeperRequestID)
		return err
	}
	return broker.Supercharger.RecordRestaurantCheck(supercharger.PlaceID, returnedIDs, !truncated, now)
}
