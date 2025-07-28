-- Schema that exactly matches the Card struct from types.go and api.txt
CREATE TABLE IF NOT EXISTS cards (
    -- Core Card Fields
    arena_id INTEGER,
    id TEXT PRIMARY KEY NOT NULL,
    lang TEXT NOT NULL,
    mtgo_id INTEGER,
    mtgo_foil_id INTEGER,
    multiverse_ids TEXT, -- JSON array of integers
    tcgplayer_id INTEGER,
    tcgplayer_etched_id INTEGER,
    cardmarket_id INTEGER,
    object TEXT NOT NULL,
    layout TEXT NOT NULL,
    oracle_id TEXT, -- Shared across all printings of the same card
    prints_search_uri TEXT NOT NULL,
    rulings_uri TEXT NOT NULL,
    scryfall_uri TEXT NOT NULL,
    uri TEXT NOT NULL,
    
    -- Gameplay Fields
    all_parts TEXT, -- JSON array of RelatedCard objects
    card_faces TEXT, -- JSON array of CardFace objects
    cmc REAL NOT NULL,
    color_identity TEXT, -- JSON array of strings
    color_indicator TEXT, -- JSON array of strings
    colors TEXT, -- JSON array of strings
    defense TEXT,
    edhrec_rank INTEGER,
    game_changer BOOLEAN,
    hand_modifier TEXT,
    keywords TEXT, -- JSON array of strings
    legalities TEXT, -- JSON object map[string]string
    life_modifier TEXT,
    loyalty TEXT,
    mana_cost TEXT,
    name TEXT NOT NULL,
    oracle_text TEXT,
    penny_rank INTEGER,
    power TEXT,
    produced_mana TEXT, -- JSON array of strings
    reserved BOOLEAN NOT NULL,
    toughness TEXT,
    type_line TEXT NOT NULL,
    
    -- Print Fields
    artist TEXT,
    artist_ids TEXT, -- JSON array of strings
    attraction_lights TEXT, -- JSON array of integers
    booster BOOLEAN NOT NULL,
    border_color TEXT NOT NULL,
    card_back_id TEXT NOT NULL,
    collector_number TEXT NOT NULL,
    content_warning BOOLEAN,
    digital BOOLEAN NOT NULL,
    finishes TEXT, -- JSON array of strings
    flavor_name TEXT,
    flavor_text TEXT,
    frame_effects TEXT, -- JSON array of strings
    frame TEXT NOT NULL,
    full_art BOOLEAN NOT NULL,
    games TEXT, -- JSON array of strings
    highres_image BOOLEAN NOT NULL,
    illustration_id TEXT,
    image_status TEXT NOT NULL,
    image_uris TEXT, -- JSON object map[string]string
    oversized BOOLEAN NOT NULL,
    prices TEXT, -- JSON object map[string]*string
    printed_name TEXT,
    printed_text TEXT,
    printed_type_line TEXT,
    promo BOOLEAN NOT NULL,
    promo_types TEXT, -- JSON array of strings
    purchase_uris TEXT, -- JSON object map[string]string
    rarity TEXT NOT NULL,
    related_uris TEXT, -- JSON object map[string]string
    released_at TEXT NOT NULL,
    reprint BOOLEAN NOT NULL,
    scryfall_set_uri TEXT NOT NULL,
    set_name TEXT NOT NULL,
    set_search_uri TEXT NOT NULL,
    set_type TEXT NOT NULL,
    set_uri TEXT NOT NULL,
    set_code TEXT NOT NULL, -- This is the "set" field in the API
    set_id TEXT NOT NULL,
    story_spotlight BOOLEAN NOT NULL,
    textless BOOLEAN NOT NULL,
    variation BOOLEAN NOT NULL,
    variation_of TEXT,
    security_stamp TEXT,
    watermark TEXT,
    preview TEXT -- JSON object CardPreview
);

-- Index on oracle_id to efficiently group printings of the same card
CREATE INDEX IF NOT EXISTS idx_cards_oracle_id ON cards(oracle_id);

-- Index on name for searching
CREATE INDEX IF NOT EXISTS idx_cards_name ON cards(name);

-- Index on set_code for set-based queries
CREATE INDEX IF NOT EXISTS idx_cards_set_code ON cards(set_code);

-- Index on rarity for filtering
CREATE INDEX IF NOT EXISTS idx_cards_rarity ON cards(rarity);

-- Index on games for platform filtering
CREATE INDEX IF NOT EXISTS idx_cards_games ON cards(games);