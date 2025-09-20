// Static mapping of restaurant primary_type to emojis
// Generated based on database query of all restaurant types in the system

export const RESTAURANT_TYPE_TO_EMOJI: Record<string, string> = {
    // Common restaurant types
    'restaurant': '🍽️',
    'fine_dining_restaurant': '🍽️',
    'meal_delivery': '🚚',
    'meal_takeaway': '📦',

    // Specific cuisines
    'american_restaurant': '🍔',
    'asian_restaurant': '🥢',
    'chinese_restaurant': '🥢',
    'japanese_restaurant': '🍣',
    'sushi_restaurant': '🍣',
    'korean_restaurant': '🍜',
    'thai_restaurant': '🌶️',
    'vietnamese_restaurant': '🍜',
    'indian_restaurant': '🍛',
    'indonesian_restaurant': '🍛',
    'italian_restaurant': '🍝',
    'french_restaurant': '🥖',
    'mexican_restaurant': '🌮',
    'spanish_restaurant': '🥘',
    'greek_restaurant': '🫒',
    'turkish_restaurant': '🥙',
    'lebanese_restaurant': '🥙',
    'middle_eastern_restaurant': '🥙',
    'mediterranean_restaurant': '🫒',
    'brazilian_restaurant': '🇧🇷',
    'afghani_restaurant': '🍲',

    // Fast food & quick service
    'fast_food_restaurant': '🍟',
    'hamburger_restaurant': '🍔',
    'pizza_restaurant': '🍕',
    'sandwich_shop': '🥪',
    'bagel_shop': '🥯',
    'deli': '🥪',
    'donut_shop': '🍩',

    // Specific food types
    'barbecue_restaurant': '🍖',
    'steak_house': '🥩',
    'seafood_restaurant': '🦞',
    'ramen_restaurant': '🍜',
    'vegan_restaurant': '🥗',
    'vegetarian_restaurant': '🥗',
    'buffet_restaurant': '🍽️',
    'breakfast_restaurant': '🍳',
    'brunch_restaurant': '🥐',
    'dessert_restaurant': '🍰',

    // Beverages & light food
    'cafe': '☕',
    'coffee_shop': '☕',
    'tea_house': '🍵',
    'juice_shop': '🧃',
    'wine_bar': '🍷',
    'bar': '🍺',
    'pub': '🍺',
    'bar_and_grill': '🍺',
    'ice_cream_shop': '🍦',

    // Bakery & sweets
    'bakery': '🥖',
    'dessert_shop': '🧁',
    'candy_store': '🍭',
    'confectionery': '🍬',
    'acai_shop': '🍓',

    // Dining venues
    'diner': '🥞',
    'food_court': '🍽️',
    'banquet_hall': '🎉',

    // Food services
    'catering_service': '🍽️',
    'food_delivery': '🚚',

    // Stores with food
    'grocery_store': '🛒',
    'supermarket': '🛒',
    'convenience_store': '🏪',
    'food_store': '🏪',
    'asian_grocery_store': '🛒',
    'butcher_shop': '🥩',
    'liquor_store': '🍾',
    'market': '🛒',
    'department_store': '🏪',
    'discount_store': '🏪',
    'drugstore': '💊',
    'warehouse_store': '🏪',
    'wholesaler': '📦',
    'food': '🍽️',

    // Non-food establishments (fallback emojis)
    'gas_station': '⛽',
    'electric_vehicle_charging_station': '🔌',
    'truck_stop': '🚛',
    'rest_stop': '🛑',
    'hotel': '🏨',
    'motel': '🏨',
    'inn': '🏨',
    'lodging': '🏨',
    'bed_and_breakfast': '🛏️',
    'guest_house': '🏠',
    'campground': '⛺',
    'rv_park': '🚐',
    'marina': '⚓',
    'ranch': '🤠',
    'farm': '🚜',
    'amusement_park': '🎢',
    'amusement_center': '🎮',
    'casino': '🎰',
    'movie_theater': '🎬',
    'bowling_alley': '🎳',
    'video_arcade': '🕹️',
    'zoo': '🦁',
    'park': '🌳',
    'tourist_attraction': '📸',
    'plaza': '🏛️',
    'point_of_interest': '📍',
    'shopping_mall': '🛍️',
    'store': '🏪',
    'gift_shop': '🎁',
    'clothing_store': '👕',
    'furniture_store': '🪑',
    'hardware_store': '🔨',
    'bicycle_store': '🚲',
    'pet_store': '🐕',
    'florist': '🌸',
    'sporting_goods_store': '⚽',
    'home_goods_store': '🏠',
    'hospital': '🏥',
    'health': '🏥',
    'pharmacy': '💊',
    'veterinary_care': '🐾',
    'gym': '💪',
    'spa': '🧘',
    'sports_club': '🏃',
    'sports_complex': '🏟️',
    'school': '🏫',
    'church': '⛪',
    'fire_station': '🚒',
    'train_station': '🚂',
    'transit_station': '🚇',
    'bus_stop': '🚌',
    'atm': '🏧',
    'courier_service': '📦',
    'consultant': '💼',
    'car_repair': '🔧',
    'child_care_agency': '👶',
    'event_venue': '🎪',
}

/**
 * Get the emoji for a restaurant type
 * @param primaryType The primary_type from the database
 * @returns The corresponding emoji or a default restaurant emoji
 */
export function getRestaurantEmoji(primaryType: string | null | undefined): string {
    if (!primaryType) return '🍽️' // Default restaurant emoji

    const emoji = RESTAURANT_TYPE_TO_EMOJI[primaryType.toLowerCase()]
    return emoji || '🍽️' // Default to restaurant emoji if not found
}
