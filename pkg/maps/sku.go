package maps

import "strings"

// determineSKU returns the Places (New) SKU based on the endpoint URL and the X-Goog-FieldMask value.
// Highest tier in the mask determines billing, per Google’s rule.
func determineSKU(endpointURL, fieldMask string) string {
	// Basic normalization
	mask := strings.ReplaceAll(fieldMask, " ", "")
	if mask == "" {
		return "unknown"
	}
	// Wildcard means "all fields" => highest tier
	if mask == "*" {
		switch {
		case endpointURL == placesAPIEndpoint:
			return "Places API Text Search Enterprise + Atmosphere"
		case endpointURL == placesNearbyEndpoint:
			return "Places API Nearby Search Enterprise + Atmosphere"
		case strings.HasPrefix(endpointURL, placeDetailsEndpoint):
			return "Places API Place Details Enterprise + Atmosphere"
		default:
			return "unknown"
		}
	}

	switch {
	case endpointURL == placesAPIEndpoint:
		return tierTextSearch(mask)
	case endpointURL == placesNearbyEndpoint:
		return tierNearbySearch(mask)
	case strings.HasPrefix(endpointURL, placeDetailsEndpoint): // e.g. https://.../v1/places/ChIJ...
		return tierPlaceDetails(mask)
	default:
		return "unknown"
	}
}

// ---------- Place Details (New) field tiers ----------
var (
	// Essentials (IDs Only)
	pdIDsOnly = set(
		"attributions", "id", "name", "photos",
	)
	// Essentials
	pdEssentials = set(
		"addressComponents", "addressDescriptor", "adrFormatAddress", "formattedAddress",
		"location", "plusCode", "postalAddress", "shortFormattedAddress", "types", "viewport",
	)
	// Pro
	pdPro = set(
		"accessibilityOptions", "businessStatus", "containingPlaces", "displayName",
		"googleMapsLinks", "googleMapsUri", "iconBackgroundColor", "iconMaskBaseUri",
		"primaryType", "primaryTypeDisplayName", "pureServiceAreaBusiness", "subDestinations",
		"utcOffsetMinutes",
	)
	// Enterprise
	pdEnt = set(
		"currentOpeningHours", "currentSecondaryOpeningHours",
		"internationalPhoneNumber", "nationalPhoneNumber",
		"priceLevel", "priceRange", "rating", "regularOpeningHours",
		"regularSecondaryOpeningHours", "userRatingCount", "websiteUri",
	)
	// Enterprise + Atmosphere
	pdEntAtmos = set(
		"allowsDogs", "curbsidePickup", "delivery", "dineIn", "editorialSummary",
		"evChargeAmenitySummary", "evChargeOptions", "fuelOptions", "generativeSummary",
		"goodForChildren", "goodForGroups", "goodForWatchingSports", "liveMusic",
		"menuForChildren", "neighborhoodSummary", "parkingOptions", "paymentOptions",
		"outdoorSeating", "reservable", "restroom", "reviews", "reviewSummary",
		// docs list routingSummaries (no prefix). Accept both for safety:
		"routingSummaries", "places.routingSummaries",
		"servesBeer", "servesBreakfast", "servesBrunch", "servesCocktails", "servesCoffee",
		"servesDessert", "servesDinner", "servesLunch", "servesVegetarianFood", "servesWine",
		"takeout",
	)
)

func tierPlaceDetails(mask string) string {
	switch {
	case hasAny(mask, pdEntAtmos):
		return "Places API Place Details Enterprise + Atmosphere"
	case hasAny(mask, pdEnt):
		return "Places API Place Details Enterprise"
	case hasAny(mask, pdPro):
		return "Places API Place Details Pro"
	case onlyIDsOrEssentials(mask, pdIDsOnly, pdEssentials):
		// If ONLY ids-only fields present, use IDs Only; otherwise Essentials.
		if hasAny(mask, pdIDsOnly) && !hasAny(mask, pdEssentials) && !hasAny(mask, pdPro) && !hasAny(mask, pdEnt) && !hasAny(mask, pdEntAtmos) {
			return "Places API Place Details Essentials (IDs Only)"
		}
		return "Places API Place Details Essentials"
	default:
		return "unknown"
	}
}

// ---------- Text Search (New) field tiers ----------
var (
	tsIDsOnly = set(
		// Text Search IDs Only uses "places." prefix plus nextPageToken
		"places.attributions", "places.id", "places.name", "nextPageToken",
	)
	tsPro = set(
		"places.accessibilityOptions", "places.addressComponents", "places.addressDescriptor",
		"places.adrFormatAddress", "places.businessStatus", "places.containingPlaces",
		"places.displayName", "places.formattedAddress", "places.googleMapsLinks",
		"places.googleMapsUri", "places.iconBackgroundColor", "places.iconMaskBaseUri",
		"places.location", "places.photos", "places.plusCode", "places.postalAddress",
		"places.primaryType", "places.primaryTypeDisplayName", "places.pureServiceAreaBusiness",
		"places.shortFormattedAddress", "places.subDestinations", "places.types",
		"places.utcOffsetMinutes", "places.viewport",
	)
	tsEnt = set(
		"places.currentOpeningHours", "places.currentSecondaryOpeningHours",
		"places.internationalPhoneNumber", "places.nationalPhoneNumber",
		"places.priceLevel", "places.priceRange", "places.rating",
		"places.regularOpeningHours", "places.regularSecondaryOpeningHours",
		"places.userRatingCount", "places.websiteUri",
	)
	tsEntAtmos = set(
		"places.allowsDogs", "places.curbsidePickup", "places.delivery", "places.dineIn",
		"places.editorialSummary", "places.evChargeAmenitySummary", "places.evChargeOptions",
		"places.fuelOptions", "places.generativeSummary", "places.goodForChildren",
		"places.goodForGroups", "places.goodForWatchingSports", "places.liveMusic",
		"places.menuForChildren", "places.neighborhoodSummary", "places.parkingOptions",
		"places.paymentOptions", "places.outdoorSeating", "places.reservable",
		"places.restroom", "places.reviews", "places.reviewSummary",
		// docs list routingSummaries (no prefix) for Search; accept both
		"routingSummaries", "places.routingSummaries",
		"places.servesBeer", "places.servesBreakfast", "places.servesBrunch",
		"places.servesCocktails", "places.servesCoffee", "places.servesDessert",
		"places.servesDinner", "places.servesLunch", "places.servesVegetarianFood",
		"places.servesWine", "places.takeout",
	)
)

func tierTextSearch(mask string) string {
	switch {
	case hasAny(mask, tsEntAtmos):
		return "Places API Text Search Enterprise + Atmosphere"
	case hasAny(mask, tsEnt):
		return "Places API Text Search Enterprise"
	case hasAny(mask, tsPro):
		return "Places API Text Search Pro"
	case onlyIDs(mask, tsIDsOnly):
		return "Places API Text Search Essentials (IDs Only)"
	default:
		return "unknown"
	}
}

// ---------- Nearby Search (New) field tiers ----------
var (
	// Nearby Search has Pro / Enterprise / Enterprise+Atmosphere (no Essentials tiers in docs)
	nbPro = set(
		"places.accessibilityOptions", "places.addressComponents", "places.addressDescriptor",
		"places.adrFormatAddress", "places.attributions", "places.businessStatus",
		"places.containingPlaces", "places.displayName", "places.formattedAddress",
		"places.googleMapsLinks", "places.googleMapsUri", "places.iconBackgroundColor",
		"places.iconMaskBaseUri", "places.id", "places.location", "places.name",
		"places.photos", "places.plusCode", "places.postalAddress", "places.primaryType",
		"places.primaryTypeDisplayName", "places.pureServiceAreaBusiness",
		"places.shortFormattedAddress", "places.subDestinations", "places.types",
		"places.utcOffsetMinutes", "places.viewport",
	)
	nbEnt = set(
		"places.currentOpeningHours", "places.currentSecondaryOpeningHours",
		"places.internationalPhoneNumber", "places.nationalPhoneNumber",
		"places.priceLevel", "places.priceRange", "places.rating",
		"places.regularOpeningHours", "places.regularSecondaryOpeningHours",
		"places.userRatingCount", "places.websiteUri",
	)
	nbEntAtmos = set(
		"places.allowsDogs", "places.curbsidePickup", "places.delivery", "places.dineIn",
		"places.editorialSummary", "places.evChargeAmenitySummary", "places.evChargeOptions",
		"places.fuelOptions", "places.generativeSummary", "places.goodForChildren",
		"places.goodForGroups", "places.goodForWatchingSports", "places.liveMusic",
		"places.menuForChildren", "places.neighborhoodSummary", "places.parkingOptions",
		"places.paymentOptions", "places.outdoorSeating", "places.reservable",
		"places.restroom", "places.reviews", "places.reviewSummary",
		// docs list routingSummaries (no prefix) for Search; accept both
		"routingSummaries", "places.routingSummaries",
		"places.servesBeer", "places.servesBreakfast", "places.servesBrunch",
		"places.servesCocktails", "places.servesCoffee", "places.servesDessert",
		"places.servesDinner", "places.servesLunch", "places.servesVegetarianFood",
		"places.servesWine", "places.takeout",
	)
)

func tierNearbySearch(mask string) string {
	switch {
	case hasAny(mask, nbEntAtmos):
		return "Places API Nearby Search Enterprise + Atmosphere"
	case hasAny(mask, nbEnt):
		return "Places API Nearby Search Enterprise"
	case hasAny(mask, nbPro):
		return "Places API Nearby Search Pro"
	default:
		return "unknown"
	}
}

// ---------- helpers ----------
func set(ss ...string) map[string]struct{} {
	m := make(map[string]struct{}, len(ss))
	for _, s := range ss {
		m[s] = struct{}{}
	}
	return m
}
func hasAny(mask string, fields map[string]struct{}) bool {
	for _, tok := range strings.Split(mask, ",") {
		if _, ok := fields[tok]; ok {
			return true
		}
		// Accept unprefixed for Details (docs list without "places.") and for special cases
		if strings.HasPrefix(tok, "places.") {
			if _, ok := fields[strings.TrimPrefix(tok, "places.")]; ok {
				return true
			}
		}
	}
	return false
}
func onlyIDs(mask string, idsOnly map[string]struct{}) bool {
	toks := strings.Split(mask, ",")
	for _, tok := range toks {
		if _, ok := idsOnly[tok]; ok {
			continue
		}
		// If not in idsOnly, fail
		return false
	}
	return true
}
func onlyIDsOrEssentials(mask string, idsOnly, essentials map[string]struct{}) bool {
	toks := strings.Split(mask, ",")
	for _, tok := range toks {
		if _, ok := idsOnly[tok]; ok {
			continue
		}
		if _, ok := essentials[tok]; ok {
			continue
		}
		// also accept "places."-prefixed forms for Details callers who include it
		if strings.HasPrefix(tok, "places.") {
			if _, ok := idsOnly[strings.TrimPrefix(tok, "places.")]; ok {
				continue
			}
			if _, ok := essentials[strings.TrimPrefix(tok, "places.")]; ok {
				continue
			}
		}
		return false
	}
	return true
}
