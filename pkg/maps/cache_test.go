package maps

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/brensch/passengerprincess/pkg/db"
	"gorm.io/gorm/logger"
)

// superchargerLocation is where the fake supercharger sits; restaurants are placed relative to it.
var superchargerLocation = Point{Latitude: 37.0, Longitude: -122.0}

// fakePlaces serves the Places API endpoints the cache uses and records the SKU of every call.
type fakePlaces struct {
	mu          sync.Mutex
	details     map[string]*PlaceDetails
	restaurants []*PlaceDetails
	// truncated makes the restaurant search report more results than it returns.
	truncated bool
	skus      []string
}

func newFakePlaces(t *testing.T) *fakePlaces {
	t.Helper()
	f := &fakePlaces{details: map[string]*PlaceDetails{}}
	srv := httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(srv.Close)

	oldSearch, oldDetails := placesAPIEndpoint, placeDetailsEndpoint
	placesAPIEndpoint = srv.URL + "/v1/places:searchText"
	placeDetailsEndpoint = srv.URL + "/v1/places"
	t.Cleanup(func() { placesAPIEndpoint, placeDetailsEndpoint = oldSearch, oldDetails })

	if err := db.Initialize(&db.Config{DatabasePath: filepath.Join(t.TempDir(), "test.db"), LogLevel: logger.Silent}); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return f
}

func (f *fakePlaces) serve(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	mask := r.Header.Get("X-Goog-FieldMask")
	idsOnly := !strings.Contains(mask, "displayName")

	if r.URL.Path == "/v1/places:searchText" {
		f.skus = append(f.skus, determineSKU(placesAPIEndpoint, mask))
		var body requestBody
		json.NewDecoder(r.Body).Decode(&body)
		page := 1
		if body.PageToken != nil {
			page, _ = strconv.Atoi(*body.PageToken)
		}
		const pageSize = 20
		start, end := (page-1)*pageSize, page*pageSize
		resp := apiResponse{}
		for i := start; i < end && i < len(f.restaurants); i++ {
			resp.Places = append(resp.Places, project(f.restaurants[i], idsOnly))
		}
		if end < len(f.restaurants) || f.truncated {
			resp.NextPageToken = strconv.Itoa(page + 1)
		}
		json.NewEncoder(w).Encode(resp)
		return
	}

	f.skus = append(f.skus, determineSKU(placeDetailsEndpoint+"/", mask))
	place, ok := f.details[strings.TrimPrefix(r.URL.Path, "/v1/places/")]
	if !ok {
		http.Error(w, `{"error":{"status":"NOT_FOUND"}}`, http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(project(place, idsOnly))
}

func project(p *PlaceDetails, idsOnly bool) *PlaceDetails {
	if idsOnly {
		return &PlaceDetails{ID: p.ID}
	}
	return p
}

func (f *fakePlaces) addSupercharger(id, name string) *PlaceDetails {
	f.mu.Lock()
	defer f.mu.Unlock()
	loc := Location(superchargerLocation)
	f.details[id] = &PlaceDetails{ID: id, DisplayName: &DisplayNameObj{Text: name}, Location: &loc}
	return f.details[id]
}

// setRestaurants sets the restaurant search results. Each place is placed metersNorth of the supercharger.
func (f *fakePlaces) setRestaurants(metersNorth map[string]float64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.restaurants = nil
	for id, meters := range metersNorth {
		loc := Location{Latitude: superchargerLocation.Latitude + meters/111320, Longitude: superchargerLocation.Longitude}
		f.restaurants = append(f.restaurants, &PlaceDetails{ID: id, DisplayName: &DisplayNameObj{Text: "Restaurant " + id}, Location: &loc})
	}
}

func (f *fakePlaces) setRestaurantStatus(id, status string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, r := range f.restaurants {
		if r.ID == id {
			r.BusinessStatus = &status
		}
	}
}

// takeCalls returns the SKUs called since the last take, split into free and paid.
func (f *fakePlaces) takeCalls() (free, paid int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, sku := range f.skus {
		if strings.Contains(sku, "IDs Only") {
			free++
		} else {
			paid++
		}
	}
	f.skus = nil
	return free, paid
}

func restaurantIDsOf(restaurants []db.RestaurantWithDistance) []string {
	out := make([]string, len(restaurants))
	for i, r := range restaurants {
		out[i] = r.PlaceID
	}
	return out
}

func storedRestaurantIDs(t *testing.T, id string) []string {
	t.Helper()
	restaurants, err := db.GetDefaultService().Supercharger.GetRestaurantsForSupercharger(id)
	if err != nil {
		t.Fatalf("GetRestaurantsForSupercharger: %v", err)
	}
	return restaurantIDsOf(restaurants)
}

func assertSameIDs(t *testing.T, got []string, want ...string) {
	t.Helper()
	set := map[string]int{}
	for _, id := range got {
		set[id]++
	}
	for _, id := range want {
		set[id]--
	}
	for _, n := range set {
		if n != 0 {
			t.Fatalf("got IDs %v, want %v", got, want)
		}
	}
}

// age moves a cached supercharger's timestamps into the past.
func age(t *testing.T, id string, updated, verified time.Duration) {
	t.Helper()
	now := time.Now().UTC()
	if err := db.DB.Exec("UPDATE superchargers SET last_updated = ?, last_verified = ? WHERE place_id = ?",
		now.Add(-updated), now.Add(-verified), id).Error; err != nil {
		t.Fatalf("age %s: %v", id, err)
	}
}

func getSupercharger(t *testing.T, id string) (*db.Supercharger, []db.RestaurantWithDistance) {
	t.Helper()
	sc, restaurants, err := GetSuperchargerWithCache(context.Background(), db.GetDefaultService(), "key", id, "", "test")
	if err != nil {
		t.Fatalf("GetSuperchargerWithCache(%s): %v", id, err)
	}
	return sc, restaurants
}

func TestGetSuperchargerWithCacheFetchesOnMiss(t *testing.T) {
	f := newFakePlaces(t)
	f.addSupercharger("sc1", "Tesla Supercharger")
	f.setRestaurants(map[string]float64{"near": 100, "far": 600, "closed": 50})
	f.setRestaurantStatus("closed", "CLOSED_PERMANENTLY")

	sc, restaurants := getSupercharger(t, "sc1")

	if sc == nil {
		t.Fatal("supercharger = nil, want sc1")
	}
	assertSameIDs(t, restaurantIDsOf(restaurants), "near")
	assertSameIDs(t, storedRestaurantIDs(t, "sc1"), "near")
	stored, err := db.GetDefaultService().Supercharger.GetByID("sc1")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	assertSameIDs(t, stored.RestaurantSearchIDs, "near", "far", "closed")
	if !stored.IsFresh(time.Now()) {
		t.Errorf("stored supercharger is not fresh: updated %v, verified %v", stored.LastUpdated, stored.LastVerified)
	}
	if _, paid := f.takeCalls(); paid != 2 {
		t.Errorf("paid calls = %d, want details and one search page", paid)
	}
}

func TestGetSuperchargerWithCacheServesFreshRowsWithoutCalls(t *testing.T) {
	f := newFakePlaces(t)
	f.addSupercharger("sc1", "Tesla Supercharger")
	f.setRestaurants(map[string]float64{"near": 100})
	getSupercharger(t, "sc1")
	f.takeCalls()

	sc, restaurants := getSupercharger(t, "sc1")

	if sc == nil {
		t.Fatal("supercharger = nil, want sc1")
	}
	assertSameIDs(t, restaurantIDsOf(restaurants), "near")
	if free, paid := f.takeCalls(); free+paid != 0 {
		t.Errorf("API calls = %d, want none", free+paid)
	}
}

func TestGetSuperchargerWithCacheRefetchesStaleRows(t *testing.T) {
	for name, stale := range map[string]struct{ updated, verified time.Duration }{
		"unverified":      {updated: time.Hour, verified: db.MaxVerifyAge + time.Minute},
		"content expired": {updated: db.MaxContentAge + time.Minute, verified: time.Hour},
	} {
		t.Run(name, func(t *testing.T) {
			f := newFakePlaces(t)
			f.addSupercharger("sc1", "Tesla Supercharger")
			f.setRestaurants(map[string]float64{"old": 100})
			getSupercharger(t, "sc1")
			age(t, "sc1", stale.updated, stale.verified)
			f.setRestaurants(map[string]float64{"new": 100})
			f.takeCalls()

			_, restaurants := getSupercharger(t, "sc1")

			assertSameIDs(t, restaurantIDsOf(restaurants), "new")
			assertSameIDs(t, storedRestaurantIDs(t, "sc1"), "new")
			if _, paid := f.takeCalls(); paid == 0 {
				t.Error("stale supercharger was served without re-fetching")
			}
		})
	}
}

func TestGetSuperchargerWithCacheRemembersNonSuperchargers(t *testing.T) {
	f := newFakePlaces(t)
	f.addSupercharger("svc", "Tesla Service Center")

	for i := 0; i < 2; i++ {
		if sc, _ := getSupercharger(t, "svc"); sc != nil {
			t.Fatalf("call %d: supercharger = %v, want nil", i, sc)
		}
	}

	if _, paid := f.takeCalls(); paid != 1 {
		t.Errorf("paid calls = %d, want one details call", paid)
	}
	if isNon, _ := db.GetDefaultService().Supercharger.IsNonSupercharger("svc"); !isNon {
		t.Error("place was not recorded as a non-supercharger")
	}
}

func TestGetSuperchargerWithCacheDropsGoneSuperchargers(t *testing.T) {
	for name, remove := range map[string]func(*fakePlaces){
		"not found": func(f *fakePlaces) { delete(f.details, "sc1") },
		"closed": func(f *fakePlaces) {
			status := "CLOSED_PERMANENTLY"
			f.details["sc1"].BusinessStatus = &status
		},
	} {
		t.Run(name, func(t *testing.T) {
			f := newFakePlaces(t)
			f.addSupercharger("sc1", "Tesla Supercharger")
			f.setRestaurants(map[string]float64{"near": 100})
			getSupercharger(t, "sc1")
			age(t, "sc1", time.Hour, db.MaxVerifyAge+time.Minute)
			remove(f)

			if sc, restaurants := getSupercharger(t, "sc1"); sc != nil || len(restaurants) != 0 {
				t.Fatalf("got %v with %d restaurants, want nothing", sc, len(restaurants))
			}
			if _, err := db.GetDefaultService().Supercharger.GetByID("sc1"); err == nil {
				t.Error("supercharger row was not deleted")
			}
			if restaurants := storedRestaurantIDs(t, "sc1"); len(restaurants) != 0 {
				t.Errorf("restaurants still linked: %v", restaurants)
			}
		})
	}
}

func TestSweepVerifiesWithFreeCallsWhenUnchanged(t *testing.T) {
	f := newFakePlaces(t)
	f.addSupercharger("sc1", "Tesla Supercharger")
	f.setRestaurants(map[string]float64{"near": 100, "far": 600})
	getSupercharger(t, "sc1")
	age(t, "sc1", 2*VerifyInterval, VerifyInterval+time.Minute)
	f.takeCalls()

	now := time.Now()
	Sweep(context.Background(), db.GetDefaultService(), "key", now)

	free, paid := f.takeCalls()
	if paid != 0 || free != 2 {
		t.Errorf("calls = %d free, %d paid; want 2 free (details and search), 0 paid", free, paid)
	}
	stored, _ := db.GetDefaultService().Supercharger.GetByID("sc1")
	if stored.LastVerified.Sub(now).Abs() > time.Millisecond {
		t.Errorf("last_verified = %v, want %v", stored.LastVerified, now)
	}
	assertSameIDs(t, storedRestaurantIDs(t, "sc1"), "near")
}

func TestSweepUnlinksRestaurantsMissingFromCompleteSearch(t *testing.T) {
	f := newFakePlaces(t)
	f.addSupercharger("sc1", "Tesla Supercharger")
	f.setRestaurants(map[string]float64{"kept": 100, "gone": 200})
	getSupercharger(t, "sc1")
	age(t, "sc1", 2*VerifyInterval, VerifyInterval+time.Minute)
	f.setRestaurants(map[string]float64{"kept": 100})
	f.takeCalls()

	Sweep(context.Background(), db.GetDefaultService(), "key", time.Now())

	assertSameIDs(t, storedRestaurantIDs(t, "sc1"), "kept")
	if _, paid := f.takeCalls(); paid != 0 {
		t.Errorf("paid calls = %d, want 0", paid)
	}
}

func TestSweepFetchesNewRestaurants(t *testing.T) {
	f := newFakePlaces(t)
	f.addSupercharger("sc1", "Tesla Supercharger")
	f.setRestaurants(map[string]float64{"kept": 100})
	getSupercharger(t, "sc1")
	age(t, "sc1", 2*VerifyInterval, VerifyInterval+time.Minute)
	f.setRestaurants(map[string]float64{"kept": 100, "opened": 150})
	f.takeCalls()

	Sweep(context.Background(), db.GetDefaultService(), "key", time.Now())

	assertSameIDs(t, storedRestaurantIDs(t, "sc1"), "kept", "opened")
	if _, paid := f.takeCalls(); paid != 1 {
		t.Errorf("paid calls = %d, want one restaurant search page", paid)
	}
}

func TestSweepTrustsTruncatedSearchesOnlyForRepeatedMisses(t *testing.T) {
	f := newFakePlaces(t)
	f.addSupercharger("sc1", "Tesla Supercharger")
	f.setRestaurants(map[string]float64{"kept": 100, "gone": 200})
	getSupercharger(t, "sc1")
	f.setRestaurants(map[string]float64{"kept": 100, "unseen": 150})
	f.truncated = true
	f.takeCalls()

	for _, want := range [][]string{{"kept", "gone"}, {"kept"}} {
		age(t, "sc1", 2*VerifyInterval, VerifyInterval+time.Minute)
		Sweep(context.Background(), db.GetDefaultService(), "key", time.Now())
		assertSameIDs(t, storedRestaurantIDs(t, "sc1"), want...)
	}
	if _, paid := f.takeCalls(); paid != 0 {
		t.Errorf("paid calls = %d, want 0 since a truncated search cannot show a place is new", paid)
	}
}

func TestSweepDeletesSuperchargersThatNoLongerExist(t *testing.T) {
	f := newFakePlaces(t)
	f.addSupercharger("sc1", "Tesla Supercharger")
	f.setRestaurants(map[string]float64{"near": 100})
	getSupercharger(t, "sc1")
	age(t, "sc1", 2*VerifyInterval, VerifyInterval+time.Minute)
	delete(f.details, "sc1")

	Sweep(context.Background(), db.GetDefaultService(), "key", time.Now())

	if _, err := db.GetDefaultService().Supercharger.GetByID("sc1"); err == nil {
		t.Error("supercharger row was not deleted")
	}
}

func TestSweepRefreshesAgingContent(t *testing.T) {
	f := newFakePlaces(t)
	sc := f.addSupercharger("sc1", "Tesla Supercharger")
	f.setRestaurants(map[string]float64{"near": 100})
	getSupercharger(t, "sc1")
	age(t, "sc1", RefreshAge+time.Minute, time.Hour)
	sc.DisplayName.Text = "Tesla Supercharger (renamed)"
	f.takeCalls()

	now := time.Now()
	Sweep(context.Background(), db.GetDefaultService(), "key", now)

	stored, _ := db.GetDefaultService().Supercharger.GetByID("sc1")
	if stored.Name != "Tesla Supercharger (renamed)" {
		t.Errorf("name = %q, want refreshed name", stored.Name)
	}
	if now.Sub(stored.LastUpdated) > time.Minute {
		t.Errorf("last_updated = %v, want about %v", stored.LastUpdated, now)
	}
	if free, paid := f.takeCalls(); paid != 2 || free != 0 {
		t.Errorf("calls = %d free, %d paid; want the refresh only, with no separate verification", free, paid)
	}
}
