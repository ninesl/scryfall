package main

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/ninesl/scryfall-api/scryfall"
	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var ddl string

const (
	APIBaseURL       = "https://api.scryfall.com"
	DefaultUserAgent = "MTGScryfallClient/1.0"
	DefaultAccept    = "application/json;q=0.9,*/*;q=0.8"
)

var (
	DefaultClientOptions = ClientOptions{
		APIURL:    APIBaseURL,
		UserAgent: DefaultUserAgent,
		Accept:    DefaultAccept,
		Client:    &http.Client{},
	}
)

type Client struct {
	baseURL   string
	userAgent string
	accept    string
	client    *http.Client
	db        *sql.DB
}

type ClientOptions struct {
	APIURL    string       // default is "https://api.scryfall.com"
	UserAgent string       // API docs recomend "{AppName}/1.0"
	Accept    string       // "application/json;q=0.9,*/*;q=0.8". could be used to take csv? TODO:
	Client    *http.Client // any http client can be used
}

// Uses DefaultClientOptions
func NewClient(appName string) (*Client, error) {
	DefaultClientOptions.UserAgent = fmt.Sprintf("%s/1.0", strings.TrimSpace(appName))
	return NewClientWithOptions(DefaultClientOptions)
}

func NewClientWithOptions(co ClientOptions) (*Client, error) {
	// Initialize database
	db, err := sql.Open("sqlite", "scryfall.db")
	if err != nil {
		return nil, err
	}

	// Create tables if they don't exist
	if _, err := db.Exec(ddl); err != nil {
		db.Close()
		return nil, err
	}

	return &Client{
		baseURL:   co.APIURL,
		userAgent: co.UserAgent,
		accept:    co.Accept,
		client:    co.Client,
		db:        db,
	}, nil
}

func (c *Client) makeRequest(endpoint string, result interface{}) error {
	fullURL := c.baseURL + endpoint

	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", c.accept)

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	return json.NewDecoder(resp.Body).Decode(result)
}

func (c *Client) getCard(id string) (*Card, error) {
	var card Card
	err := c.makeRequest("/cards/"+url.PathEscape(id), &card)
	return &card, err
}

func (c *Client) getSet(code string) (*Set, error) {
	var set Set
	err := c.makeRequest("/sets/"+url.PathEscape(code), &set)
	return &set, err
}

func (c *Client) searchCards(query string) (*List, error) {
	var list List
	err := c.makeRequest("/cards/search?q="+url.QueryEscape(query), &list)
	return &list, err
}

func (c *Client) searchCardsByName(name string) (*List, error) {
	var list List
	query := "!\"" + name + "\""
	err := c.makeRequest("/cards/search?q="+url.QueryEscape(query), &list)
	return &list, err
}

func (c *Client) getCardPrintings(printsSearchURI string) (*List, error) {
	var list List
	// Extract the path from the full URI
	parsedURL, err := url.Parse(printsSearchURI)
	if err != nil {
		return nil, err
	}
	endpoint := parsedURL.Path + "?" + parsedURL.RawQuery
	err = c.makeRequest(endpoint, &list)
	return &list, err
}

// Helper functions

// Helper function to convert int slice to comma-separated string
func intsToString(ints []int) string {
	if len(ints) == 0 {
		return ""
	}
	strs := make([]string, len(ints))
	for i, v := range ints {
		strs[i] = strconv.Itoa(v)
	}
	return strings.Join(strs, ",")
}

// Helper function to convert pointer to sql.NullString
func ptrToNullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: *s, Valid: true}
}

// Helper function to convert pointer to sql.NullInt64
func ptrToNullInt64(i *int) sql.NullInt64 {
	if i == nil {
		return sql.NullInt64{Valid: false}
	}
	return sql.NullInt64{Int64: int64(*i), Valid: true}
}

// Helper function to convert pointer to sql.NullBool
func ptrToNullBool(b *bool) sql.NullBool {
	if b == nil {
		return sql.NullBool{Valid: false}
	}
	return sql.NullBool{Bool: *b, Valid: true}
}

// Helper function to convert string to sql.NullString
func stringToNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}

// Helper function to convert any value to JSON string
func toJSONString(v interface{}) sql.NullString {
	if v == nil {
		return sql.NullString{Valid: false}
	}

	// Handle empty slices and maps
	switch val := v.(type) {
	case []string:
		if len(val) == 0 {
			return sql.NullString{Valid: false}
		}
	case []int:
		if len(val) == 0 {
			return sql.NullString{Valid: false}
		}
	case map[string]string:
		if len(val) == 0 {
			return sql.NullString{Valid: false}
		}
	case map[string]*string:
		if len(val) == 0 {
			return sql.NullString{Valid: false}
		}
	}

	jsonBytes, err := json.Marshal(v)
	if err != nil {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: string(jsonBytes), Valid: true}
}

// toJSONStringDirect converts interface{} to JSON string directly (not sql.NullString)
func toJSONStringDirect(v interface{}) string {
	if v == nil {
		return "[]"
	}

	// Handle empty slices and maps
	switch val := v.(type) {
	case []string:
		if len(val) == 0 {
			return "[]"
		}
	case []int:
		if len(val) == 0 {
			return "[]"
		}
	case map[string]string:
		if len(val) == 0 {
			return "{}"
		}
	case map[string]*string:
		if len(val) == 0 {
			return "{}"
		}
	}

	jsonBytes, err := json.Marshal(v)
	if err != nil {
		return "[]"
	}
	return string(jsonBytes)
}

// containsFinish checks if a finish type exists in the finishes array
func containsFinish(finishes []string, finish string) bool {
	for _, f := range finishes {
		if f == finish {
			return true
		}
	}
	return false
}

func isArenaSet(games []string) bool {
	for _, game := range games {
		if game == "arena" {
			return true
		}
	}
	return false
}

func shouldIncludeCard(printings []Card) bool {
	// Check if any printing is common/uncommon on Arena
	for _, printing := range printings {
		if isArenaSet(printing.Games) && (printing.Rarity == "common" || printing.Rarity == "uncommon") {
			return false
		}
	}
	return true
}

// queryAndInsertCards fetches cards from Scryfall API and inserts them into database
func (c *Client) queryAndInsertCards(db *sql.DB) error {
	ctx := context.Background()
	queries := scryfall.New(db)

	searchQuery := "(game:paper game:mtgo -game:arena in:common or in:uncommon) game:arena r>=rare"
	fmt.Printf("Searching for query: %s\n", searchQuery)

	results, err := c.searchCards(searchQuery)
	if err != nil {
		return fmt.Errorf("search error: %v", err)
	}

	fmt.Printf("Found %d cards\n", results.TotalCards)

	insertedCount := 0
	for _, card := range results.Data {
		fmt.Printf("Fetching printings for %s...\n", card.Name)

		printings, err := c.getCardPrintings(card.PrintsSearchURI.String())
		if err != nil {
			log.Printf("Error fetching printings for %s: %v", card.Name, err)
			continue
		}

		// Filter out cards that have common/uncommon Arena printings
		if !shouldIncludeCard(printings.Data) {
			fmt.Printf("Skipping %s - has common/uncommon Arena printing\n", card.Name)
			continue
		}

		// First, insert the card (oracle-level data) - this will be upserted if it already exists
		err = queries.UpsertCard(ctx, scryfall.UpsertCardParams{
			OracleID:        *card.OracleID,
			Name:            card.Name,
			Layout:          card.Layout,
			PrintsSearchUri: card.PrintsSearchURI.String(),
			RulingsUri:      card.RulingsURI.String(),
			AllParts:        toJSONString(card.AllParts),
			CardFaces:       toJSONString(card.CardFaces),
			Cmc:             card.CMC,
			ColorIdentity:   toJSONStringDirect(card.ColorIdentity),
			ColorIndicator:  toJSONString(card.ColorIndicator),
			Colors:          toJSONString(card.Colors),
			Defense:         ptrToNullString(card.Defense),
			EdhrecRank:      ptrToNullInt64(card.EDHRecRank),
			GameChanger:     ptrToNullBool(card.GameChanger),
			HandModifier:    ptrToNullString(card.HandModifier),
			Keywords:        toJSONStringDirect(card.Keywords),
			Legalities:      toJSONStringDirect(card.Legalities),
			LifeModifier:    ptrToNullString(card.LifeModifier),
			Loyalty:         ptrToNullString(card.Loyalty),
			ManaCost:        ptrToNullString(card.ManaCost),
			OracleText:      ptrToNullString(card.OracleText),
			PennyRank:       ptrToNullInt64(card.PennyRank),
			Power:           ptrToNullString(card.Power),
			ProducedMana:    toJSONString(card.ProducedMana),
			Reserved:        card.Reserved,
			Toughness:       ptrToNullString(card.Toughness),
			TypeLine:        card.TypeLine,
		})

		if err != nil {
			log.Printf("Error inserting card %s: %v", card.Name, err)
			continue
		}

		// Then insert ALL printings of this card
		for _, printing := range printings.Data {
			err = queries.UpsertPrinting(ctx, scryfall.UpsertPrintingParams{
				ID:                printing.ID,
				OracleID:          *printing.OracleID,
				ArenaID:           ptrToNullInt64(printing.ArenaID),
				Lang:              printing.Lang,
				MtgoID:            ptrToNullInt64(printing.MTGOID),
				MtgoFoilID:        ptrToNullInt64(printing.MTGOFoilID),
				MultiverseIds:     toJSONString(printing.MultiverseIDs),
				TcgplayerID:       ptrToNullInt64(printing.TCGPlayerID),
				TcgplayerEtchedID: ptrToNullInt64(printing.TCGPlayerEtchedID),
				CardmarketID:      ptrToNullInt64(printing.CardmarketID),
				Object:            printing.Object,
				ScryfallUri:       printing.ScryfallURI.String(),
				Uri:               printing.URI.String(),
				Artist:            ptrToNullString(printing.Artist),
				ArtistIds:         toJSONString(printing.ArtistIDs),
				AttractionLights:  toJSONString(printing.AttractionLights),
				Booster:           printing.Booster,
				BorderColor:       printing.BorderColor,
				CardBackID:        printing.CardBackID,
				CollectorNumber:   printing.CollectorNumber,
				ContentWarning:    ptrToNullBool(printing.ContentWarning),
				Digital:           printing.Digital,
				Finishes:          toJSONStringDirect(printing.Finishes),
				FlavorName:        ptrToNullString(printing.FlavorName),
				FlavorText:        ptrToNullString(printing.FlavorText),
				Foil:              containsFinish(printing.Finishes, "foil"),
				Nonfoil:           containsFinish(printing.Finishes, "nonfoil"),
				FrameEffects:      toJSONString(printing.FrameEffects),
				Frame:             printing.Frame,
				FullArt:           printing.FullArt,
				Games:             toJSONStringDirect(printing.Games),
				HighresImage:      printing.HighresImage,
				IllustrationID:    ptrToNullString(printing.IllustrationID),
				ImageStatus:       printing.ImageStatus,
				ImageUris:         toJSONString(printing.ImageURIs),
				Oversized:         printing.Oversized,
				Prices:            toJSONStringDirect(printing.Prices),
				PrintedName:       ptrToNullString(printing.PrintedName),
				PrintedText:       ptrToNullString(printing.PrintedText),
				PrintedTypeLine:   ptrToNullString(printing.PrintedTypeLine),
				Promo:             printing.Promo,
				PromoTypes:        toJSONString(printing.PromoTypes),
				PurchaseUris:      toJSONString(printing.PurchaseURIs),
				Rarity:            printing.Rarity,
				RelatedUris:       toJSONStringDirect(printing.RelatedURIs),
				ReleasedAt:        printing.ReleasedAt,
				Reprint:           printing.Reprint,
				ScryfallSetUri:    printing.ScryfallSetURI.String(),
				SetName:           printing.SetName,
				SetSearchUri:      printing.SetSearchURI.String(),
				SetType:           printing.SetType,
				SetUri:            printing.SetURI.String(),
				Set:               printing.Set,
				SetID:             printing.SetID,
				StorySpotlight:    printing.StorySpotlight,
				Textless:          printing.Textless,
				Variation:         printing.Variation,
				VariationOf:       ptrToNullString(printing.VariationOf),
				SecurityStamp:     ptrToNullString(printing.SecurityStamp),
				Watermark:         ptrToNullString(printing.Watermark),
				Preview:           toJSONString(printing.Preview),
			})

			if err != nil {
				log.Printf("Error inserting printing %s (%s): %v", printing.Name, printing.Set, err)
				continue
			}

			insertedCount++
			fmt.Printf("Inserted %s (%s - %s)\n", printing.Name, printing.Set, printing.Rarity)
		}
	}

	fmt.Printf("\nInserted %d filtered cards into database\n", insertedCount)
	return nil
}

// loadCardsFromDatabase loads cards from database and returns them as []Card with printings grouped
func (c *Client) loadCardsFromDatabase(db *sql.DB) ([]Card, error) {
	ctx := context.Background()
	queries := scryfall.New(db)

	cardPrintings, err := queries.GetCardsWithPrintings(ctx)
	if err != nil {
		return nil, fmt.Errorf("error loading cards: %v", err)
	}

	// Group printings by oracle_id to create unique cards
	cardMap := make(map[string]*Card)

	for _, row := range cardPrintings {
		// Check if we already have this card
		if existingCard, exists := cardMap[row.OracleID]; exists {
			// Add this printing's games to the existing card's games
			if row.Games != "" {
				var printingGames []string
				json.Unmarshal([]byte(row.Games), &printingGames)

				// Merge games without duplicates
				gameSet := make(map[string]bool)
				for _, game := range existingCard.Games {
					gameSet[game] = true
				}
				for _, game := range printingGames {
					gameSet[game] = true
				}

				// Convert back to slice
				var allGames []string
				for game := range gameSet {
					allGames = append(allGames, game)
				}
				existingCard.Games = allGames
			}
		} else {
			// Create new card entry
			card := Card{
				ID:       row.OracleID, // Use oracle_id as the main ID for the card
				Name:     row.Name,
				Layout:   row.Layout,
				OracleID: &row.OracleID,
				CMC:      row.Cmc,
				TypeLine: row.TypeLine,
			}

			// Handle nullable fields
			if row.ManaCost.Valid {
				card.ManaCost = &row.ManaCost.String
			}
			if row.OracleText.Valid {
				card.OracleText = &row.OracleText.String
			}

			// Parse JSON fields
			if row.Games != "" {
				json.Unmarshal([]byte(row.Games), &card.Games)
			}
			if row.ColorIdentity != "" {
				json.Unmarshal([]byte(row.ColorIdentity), &card.ColorIdentity)
			}
			if row.Colors.Valid && row.Colors.String != "" {
				json.Unmarshal([]byte(row.Colors.String), &card.Colors)
			}

			cardMap[row.OracleID] = &card
		}
	}

	// Convert map to slice
	var cards []Card
	for _, card := range cardMap {
		cards = append(cards, *card)
	}

	return cards, nil
}

// SearchCardsByQuery searches Scryfall API and returns just the cards (not the List wrapper)
func (c *Client) SearchCardsByQuery(query string) ([]Card, error) {
	list, err := c.searchCards(query)
	if err != nil {
		return nil, err
	}
	return list.Data, nil
}

// FetchFilteredScryfallAPI fetches filtered cards from Scryfall API and populates the database
func (c *Client) FetchFilteredScryfallAPI() error {
	return c.queryAndInsertCards(c.db)
}

// GetFilteredCards returns all filtered cards from the database as []Card
func (c *Client) GetFilteredCards() ([]Card, error) {
	return c.loadCardsFromDatabase(c.db)
}
