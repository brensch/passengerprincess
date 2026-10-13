package db

import (
	"path/filepath"
	"sort"
	"testing"
	"time"

	"gorm.io/gorm/logger"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	if err := Initialize(&Config{DatabasePath: path, LogLevel: logger.Silent}); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	t.Cleanup(func() { Close() })
	return GetDefaultService()
}

func testSupercharger(id string, updated, verified time.Time) *Supercharger {
	return &Supercharger{
		PlaceID:      id,
		Name:         "Tesla Supercharger " + id,
		Latitude:     10,
		Longitude:    20,
		LastUpdated:  updated,
		LastVerified: verified,
	}
}

func testRestaurant(id string, distance float64, updated time.Time) RestaurantWithDistance {
	return RestaurantWithDistance{
		Restaurant: Restaurant{PlaceID: id, Name: "Restaurant " + id, Latitude: 10, Longitude: 20, LastUpdated: updated},
		Distance:   distance,
	}
}

func restaurantIDs(t *testing.T, s *Service, superchargerID string) []string {
	t.Helper()
	restaurants, err := s.Supercharger.GetRestaurantsForSupercharger(superchargerID)
	if err != nil {
		t.Fatalf("GetRestaurantsForSupercharger: %v", err)
	}
	ids := make([]string, len(restaurants))
	for i, r := range restaurants {
		ids[i] = r.PlaceID
	}
	return ids
}

func restaurantExists(t *testing.T, s *Service, id string) bool {
	t.Helper()
	var count int64
	if err := s.db.Model(&Restaurant{}).Where("place_id = ?", id).Count(&count).Error; err != nil {
		t.Fatalf("count restaurants: %v", err)
	}
	return count > 0
}

func assertIDs(t *testing.T, got []string, want ...string) {
	t.Helper()
	sort.Strings(got)
	sort.Strings(want)
	if len(got) != len(want) {
		t.Fatalf("got IDs %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("got IDs %v, want %v", got, want)
		}
	}
}

func TestSaveWithRestaurantsReplacesRestaurants(t *testing.T) {
	s := newTestService(t)
	now := time.Now()

	if err := s.Supercharger.SaveWithRestaurants(testSupercharger("sc1", now, now), []RestaurantWithDistance{
		testRestaurant("shared", 100, now),
		testRestaurant("gone", 200, now),
	}); err != nil {
		t.Fatalf("save sc1: %v", err)
	}
	if err := s.Supercharger.SaveWithRestaurants(testSupercharger("sc2", now, now), []RestaurantWithDistance{
		testRestaurant("shared", 300, now),
	}); err != nil {
		t.Fatalf("save sc2: %v", err)
	}

	later := now.Add(time.Hour)
	renamed := testRestaurant("shared", 150, later)
	renamed.Name = "Renamed"
	sc1 := testSupercharger("sc1", later, later)
	sc1.Name = "Renamed Supercharger"
	sc1.RestaurantSearchIDs = []string{"shared", "far-away"}
	if err := s.Supercharger.SaveWithRestaurants(sc1, []RestaurantWithDistance{renamed, testRestaurant("new", 50, now)}); err != nil {
		t.Fatalf("resave sc1: %v", err)
	}

	assertIDs(t, restaurantIDs(t, s, "sc1"), "shared", "new")
	assertIDs(t, restaurantIDs(t, s, "sc2"), "shared")
	if restaurantExists(t, s, "gone") {
		t.Error("restaurant linked to no supercharger was not deleted")
	}

	got, err := s.Supercharger.GetByID("sc1")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "Renamed Supercharger" || got.LastUpdated.Sub(later).Abs() > time.Millisecond {
		t.Errorf("supercharger = %q updated %v, want updated name and time %v", got.Name, got.LastUpdated, later)
	}
	assertIDs(t, got.RestaurantSearchIDs, "shared", "far-away")

	restaurants, err := s.Supercharger.GetRestaurantsForSupercharger("sc2")
	if err != nil {
		t.Fatalf("GetRestaurantsForSupercharger: %v", err)
	}
	if restaurants[0].Name != "Renamed" || restaurants[0].Distance != 300 || restaurants[0].LastUpdated.Sub(later).Abs() > time.Millisecond {
		t.Errorf("shared restaurant = %q at %.0fm updated %v, want updated name and time at sc2's distance", restaurants[0].Name, restaurants[0].Distance, restaurants[0].LastUpdated)
	}
}

func TestRecordRestaurantCheckCompleteSearchUnlinksMissing(t *testing.T) {
	s := newTestService(t)
	old := time.Now().Add(-48 * time.Hour)
	if err := s.Supercharger.SaveWithRestaurants(testSupercharger("sc1", old, old), []RestaurantWithDistance{
		testRestaurant("kept", 100, old),
		testRestaurant("closed", 200, old),
	}); err != nil {
		t.Fatalf("save: %v", err)
	}

	checkedAt := time.Now()
	if err := s.Supercharger.RecordRestaurantCheck("sc1", []string{"kept", "unrelated"}, true, checkedAt); err != nil {
		t.Fatalf("RecordRestaurantCheck: %v", err)
	}

	assertIDs(t, restaurantIDs(t, s, "sc1"), "kept")
	if restaurantExists(t, s, "closed") {
		t.Error("unlinked restaurant was not deleted")
	}
	got, _ := s.Supercharger.GetByID("sc1")
	if got.LastVerified.Sub(checkedAt).Abs() > time.Millisecond {
		t.Errorf("last_verified = %v, want %v", got.LastVerified, checkedAt)
	}
	if got.LastUpdated.Sub(old).Abs() > time.Millisecond {
		t.Errorf("last_updated changed to %v, want %v", got.LastUpdated, old)
	}
}

func TestRecordRestaurantCheckTruncatedSearchNeedsConsecutiveMisses(t *testing.T) {
	s := newTestService(t)
	now := time.Now()
	if err := s.Supercharger.SaveWithRestaurants(testSupercharger("sc1", now, now), []RestaurantWithDistance{
		testRestaurant("flaky", 100, now),
		testRestaurant("missing", 200, now),
	}); err != nil {
		t.Fatalf("save: %v", err)
	}

	checks := []struct {
		returned []string
		want     []string
	}{
		{returned: []string{}, want: []string{"flaky", "missing"}},
		{returned: []string{"flaky"}, want: []string{"flaky"}},
		{returned: []string{}, want: []string{"flaky"}},
		{returned: []string{}, want: []string{}},
	}
	for i, check := range checks {
		if err := s.Supercharger.RecordRestaurantCheck("sc1", check.returned, false, now); err != nil {
			t.Fatalf("check %d: %v", i, err)
		}
		assertIDs(t, restaurantIDs(t, s, "sc1"), check.want...)
	}
}

func TestGetFreshByLocationHidesStaleSuperchargers(t *testing.T) {
	s := newTestService(t)
	// Rows are written in one zone and queried in another, since timestamps are compared as text.
	written := time.Now().In(time.FixedZone("UTC-8", -8*60*60))
	now := written.In(time.FixedZone("UTC+10", 10*60*60))

	superchargers := map[string]*Supercharger{
		"fresh":           testSupercharger("fresh", written.Add(-29*24*time.Hour), written.Add(-71*time.Hour)),
		"unverified":      testSupercharger("unverified", written, written.Add(-73*time.Hour)),
		"content-expired": testSupercharger("content-expired", written.Add(-31*24*time.Hour), written),
	}
	for id, sc := range superchargers {
		if err := s.Supercharger.SaveWithRestaurants(sc, []RestaurantWithDistance{testRestaurant("near-"+id, 10, now)}); err != nil {
			t.Fatalf("save %s: %v", id, err)
		}
	}
	// A restaurant shared with a fresh supercharger stays visible.
	if err := s.Supercharger.SaveWithRestaurants(superchargers["fresh"], []RestaurantWithDistance{
		testRestaurant("near-fresh", 10, now),
		testRestaurant("near-unverified", 10, now),
	}); err != nil {
		t.Fatalf("resave fresh: %v", err)
	}

	gotSuperchargers, err := s.Supercharger.GetFreshByLocation(0, 90, 0, 90, now)
	if err != nil {
		t.Fatalf("Supercharger.GetFreshByLocation: %v", err)
	}
	var ids []string
	for _, sc := range gotSuperchargers {
		ids = append(ids, sc.PlaceID)
	}
	assertIDs(t, ids, "fresh")

	gotRestaurants, err := s.Restaurant.GetFreshByLocation(0, 90, 0, 90, now)
	if err != nil {
		t.Fatalf("Restaurant.GetFreshByLocation: %v", err)
	}
	ids = nil
	for _, r := range gotRestaurants {
		ids = append(ids, r.PlaceID)
	}
	assertIDs(t, ids, "near-fresh", "near-unverified")

	outside, err := s.Supercharger.GetFreshByLocation(50, 60, 50, 60, now)
	if err != nil || len(outside) != 0 {
		t.Errorf("superchargers outside bounds = %v, %v; want none", outside, err)
	}
}

func TestListDueOrdersOldestFirst(t *testing.T) {
	s := newTestService(t)
	now := time.Now()
	for i, id := range []string{"newest", "oldest", "middle"} {
		age := []time.Duration{time.Hour, 3 * time.Hour, 2 * time.Hour}[i]
		if err := s.Supercharger.SaveWithRestaurants(testSupercharger(id, now.Add(-age), now.Add(-age)), nil); err != nil {
			t.Fatalf("save %s: %v", id, err)
		}
	}

	for name, list := range map[string]func(time.Time, int) ([]Supercharger, error){
		"ListUpdatedBefore":  s.Supercharger.ListUpdatedBefore,
		"ListVerifiedBefore": s.Supercharger.ListVerifiedBefore,
	} {
		got, err := list(now.Add(-90*time.Minute), 10)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if len(got) != 2 || got[0].PlaceID != "oldest" || got[1].PlaceID != "middle" {
			t.Errorf("%s returned %v, want [oldest middle]", name, got)
		}
		limited, _ := list(now, 1)
		if len(limited) != 1 || limited[0].PlaceID != "oldest" {
			t.Errorf("%s with limit 1 returned %v, want [oldest]", name, limited)
		}
	}
}

func TestMarkNonSuperchargerRemovesCachedData(t *testing.T) {
	s := newTestService(t)
	now := time.Now()
	if err := s.Supercharger.SaveWithRestaurants(testSupercharger("sc1", now, now), []RestaurantWithDistance{testRestaurant("r1", 10, now)}); err != nil {
		t.Fatalf("save: %v", err)
	}

	if err := s.Supercharger.MarkNonSupercharger("sc1"); err != nil {
		t.Fatalf("MarkNonSupercharger: %v", err)
	}
	if err := s.Supercharger.MarkNonSupercharger("sc1"); err != nil {
		t.Fatalf("MarkNonSupercharger twice: %v", err)
	}

	if isNon, err := s.Supercharger.IsNonSupercharger("sc1"); err != nil || !isNon {
		t.Errorf("IsNonSupercharger = %v, %v; want true", isNon, err)
	}
	if _, err := s.Supercharger.GetByID("sc1"); err == nil {
		t.Error("supercharger row was not deleted")
	}
	if restaurantExists(t, s, "r1") {
		t.Error("orphaned restaurant was not deleted")
	}

	if err := s.Supercharger.SaveWithRestaurants(testSupercharger("sc1", now, now), nil); err != nil {
		t.Fatalf("save after mark: %v", err)
	}
	if isNon, _ := s.Supercharger.IsNonSupercharger("sc1"); isNon {
		t.Error("saving a supercharger did not clear its non-supercharger mark")
	}
}

func TestDeleteRemovesSuperchargerAndOrphans(t *testing.T) {
	s := newTestService(t)
	now := time.Now()
	if err := s.Supercharger.SaveWithRestaurants(testSupercharger("sc1", now, now), []RestaurantWithDistance{testRestaurant("only-sc1", 10, now), testRestaurant("shared", 10, now)}); err != nil {
		t.Fatalf("save sc1: %v", err)
	}
	if err := s.Supercharger.SaveWithRestaurants(testSupercharger("sc2", now, now), []RestaurantWithDistance{testRestaurant("shared", 10, now)}); err != nil {
		t.Fatalf("save sc2: %v", err)
	}

	if err := s.Supercharger.Delete("sc1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if _, err := s.Supercharger.GetByID("sc1"); err == nil {
		t.Error("supercharger row was not deleted")
	}
	if restaurantExists(t, s, "only-sc1") {
		t.Error("orphaned restaurant was not deleted")
	}
	assertIDs(t, restaurantIDs(t, s, "sc2"), "shared")
}
