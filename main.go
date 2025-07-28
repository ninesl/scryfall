package main

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ninesl/scryfall-api/scryfall"
	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var ddl string

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

// getRarityValue returns numeric value for rarity comparison (lower = more common)
func getRarityValue(rarity string) int {
	switch rarity {
	case "common":
		return 1
	case "uncommon":
		return 2
	case "rare":
		return 3
	case "mythic":
		return 4
	default:
		return 5 // special/unknown rarities
	}
}

// queryAndInsertCards fetches cards from Scryfall API and inserts them into database
func queryAndInsertCards(db *sql.DB) error {
	ctx := context.Background()
	queries := scryfall.New(db)

	// Initialize Scryfall client
	client, err := NewClient("MagicClubDB")
	if err != nil {
		return err
	}

	searchQuery := "(game:paper game:mtgo -game:arena in:common or in:uncommon) game:arena r>=rare"
	fmt.Printf("Searching for query: %s\n", searchQuery)

	results, err := client.SearchCards(searchQuery)
	if err != nil {
		return fmt.Errorf("search error: %v", err)
	}

	fmt.Printf("Found %d cards\n", results.TotalCards)

	insertedCount := 0
	for _, card := range results.Data {
		fmt.Printf("Fetching printings for %s...\n", card.Name)

		printings, err := client.GetCardPrintings(card.PrintsSearchURI.String())
		if err != nil {
			log.Printf("Error fetching printings for %s: %v", card.Name, err)
			continue
		}

		// Filter out cards that have common/uncommon Arena printings
		if !shouldIncludeCard(printings.Data) {
			fmt.Printf("Skipping %s - has common/uncommon Arena printing\n", card.Name)
			continue
		}

		// Insert ALL printings of this card into database
		for _, printing := range printings.Data {
			err = queries.UpsertCard(ctx, scryfall.UpsertCardParams{
				ArenaID:           ptrToNullInt64(printing.ArenaID),
				ID:                printing.ID,
				Lang:              printing.Lang,
				MtgoID:            ptrToNullInt64(printing.MTGOID),
				MtgoFoilID:        ptrToNullInt64(printing.MTGOFoilID),
				MultiverseIds:     stringToNullString(intsToString(printing.MultiverseIDs)),
				TcgplayerID:       ptrToNullInt64(printing.TCGPlayerID),
				TcgplayerEtchedID: ptrToNullInt64(printing.TCGPlayerEtchedID),
				CardmarketID:      ptrToNullInt64(printing.CardmarketID),
				Object:            printing.Object,
				Layout:            printing.Layout,
				OracleID:          ptrToNullString(printing.OracleID),
				PrintsSearchUri:   printing.PrintsSearchURI.String(),
				RulingsUri:        printing.RulingsURI.String(),
				ScryfallUri:       printing.ScryfallURI.String(),
				Uri:               printing.URI.String(),
				AllParts:          mapToJSONString(printing.AllParts),
				CardFaces:         mapToJSONString(printing.CardFaces),
				Cmc:               printing.CMC,
				ColorIdentity:     stringToNullString(strings.Join(printing.ColorIdentity, ",")),
				ColorIndicator:    stringToNullString(strings.Join(printing.ColorIndicator, ",")),
				Colors:            stringToNullString(strings.Join(printing.Colors, ",")),
				Defense:           sql.NullString{Valid: false}, // Not in Card struct
				EdhrecRank:        ptrToNullInt64(printing.EDHRecRank),
				GameChanger:       sql.NullBool{Valid: false}, // Not in Card struct
				HandModifier:      ptrToNullString(printing.HandModifier),
				Keywords:          stringToNullString(strings.Join(printing.Keywords, ",")),
				Legalities:        mapToJSONString(printing.Legalities),
				LifeModifier:      ptrToNullString(printing.LifeModifier),
				Loyalty:           ptrToNullString(printing.Loyalty),
				ManaCost:          ptrToNullString(printing.ManaCost),
				Name:              printing.Name,
				OracleText:        ptrToNullString(printing.OracleText),
				PennyRank:         ptrToNullInt64(printing.PennyRank),
				Power:             ptrToNullString(printing.Power),
				ProducedMana:      stringToNullString(strings.Join(printing.ProducedMana, ",")),
				Reserved:          printing.Reserved,
				Toughness:         ptrToNullString(printing.Toughness),
				TypeLine:          printing.TypeLine,
				Artist:            ptrToNullString(printing.Artist),
				ArtistIds:         stringToNullString(strings.Join(printing.ArtistIDs, ",")),
				AttractionLights:  stringToNullString(intsToString(printing.AttractionLights)),
				Booster:           printing.Booster,
				BorderColor:       printing.BorderColor,
				CardBackID:        printing.CardBackID,
				CollectorNumber:   printing.CollectorNumber,
				ContentWarning:    ptrToNullBool(printing.ContentWarning),
				Digital:           printing.Digital,
				Finishes:          stringToNullString(strings.Join(printing.Finishes, ",")),
				FlavorName:        ptrToNullString(printing.FlavorName),
				FlavorText:        ptrToNullString(printing.FlavorText),
				FrameEffects:      stringToNullString(strings.Join(printing.FrameEffects, ",")),
				Frame:             printing.Frame,
				FullArt:           printing.FullArt,
				Games:             stringToNullString(strings.Join(printing.Games, ",")),
				HighresImage:      printing.HighresImage,
				IllustrationID:    ptrToNullString(printing.IllustrationID),
				ImageStatus:       printing.ImageStatus,
				ImageUris:         mapToJSONString(printing.ImageURIs),
				Oversized:         printing.Oversized,
				Prices:            mapToJSONString(printing.Prices),
				PrintedName:       sql.NullString{Valid: false}, // Not in Card struct
				PrintedText:       sql.NullString{Valid: false}, // Not in Card struct
				PrintedTypeLine:   sql.NullString{Valid: false}, // Not in Card struct
				Promo:             printing.Promo,
				PromoTypes:        sql.NullString{Valid: false}, // Not in Card struct
				PurchaseUris:      mapToJSONString(printing.PurchaseURIs),
				Rarity:            printing.Rarity,
				RelatedUris:       mapToJSONString(printing.RelatedURIs),
				ReleasedAt:        printing.ReleasedAt,
				Reprint:           printing.Reprint,
				ScryfallSetUri:    printing.ScryfallSetURI.String(),
				SetName:           printing.SetName,
				SetSearchUri:      printing.SetSearchURI.String(),
				SetType:           printing.SetType,
				SetUri:            printing.SetURI.String(),
				SetCode:           printing.Set,
				SetID:             printing.SetID,
				StorySpotlight:    printing.StorySpotlight,
				Textless:          printing.Textless,
				Variation:         printing.Variation,
				VariationOf:       sql.NullString{Valid: false}, // Not in Card struct
				SecurityStamp:     ptrToNullString(printing.SecurityStamp),
				Watermark:         ptrToNullString(printing.Watermark),
				Preview:           mapToJSONString(printing.Preview),
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

// loadAndDisplayCards loads cards from database and displays them with all rarities per game
func loadAndDisplayCards(db *sql.DB) error {
	ctx := context.Background()
	queries := scryfall.New(db)

	cards, err := queries.GetCards(ctx)
	if err != nil {
		return fmt.Errorf("error loading cards: %v", err)
	}

	fmt.Printf("\nLoaded %d cards from database:\n\n", len(cards))

	// Print table header
	fmt.Printf("%-30s %-20s %-20s %-20s\n", "Card Name", "Paper", "MTGO", "Arena")
	fmt.Printf("%-30s %-20s %-20s %-20s\n", strings.Repeat("-", 30), strings.Repeat("-", 20), strings.Repeat("-", 20), strings.Repeat("-", 20))

	// Group cards by name to find all rarities per game
	cardsByName := make(map[string][]scryfall.Card)
	for _, card := range cards {
		cardsByName[card.Name] = append(cardsByName[card.Name], card)
	}

	// Process each unique card name
	for cardName, printings := range cardsByName {
		// Track all rarities for each game
		gameRarities := make(map[string]map[string]bool)

		for _, printing := range printings {
			if !printing.Games.Valid {
				continue
			}

			games := strings.Split(printing.Games.String, ",")
			for _, game := range games {
				game = strings.TrimSpace(game)
				if game == "" {
					continue
				}

				// Initialize map for this game if it doesn't exist
				if gameRarities[game] == nil {
					gameRarities[game] = make(map[string]bool)
				}

				// Add this rarity to the game
				gameRarities[game][printing.Rarity] = true
			}
		}

		// Format output as table with consistent column widths
		var paperRarities, mtgoRarities, arenaRarities []string

		// Extract rarities for each game
		if rarities, exists := gameRarities["paper"]; exists {
			for rarity := range rarities {
				paperRarities = append(paperRarities, getRarityAbbrev(rarity))
			}
		}
		if rarities, exists := gameRarities["mtgo"]; exists {
			for rarity := range rarities {
				mtgoRarities = append(mtgoRarities, getRarityAbbrev(rarity))
			}
		}
		if rarities, exists := gameRarities["arena"]; exists {
			for rarity := range rarities {
				arenaRarities = append(arenaRarities, getRarityAbbrev(rarity))
			}
		}

		// Format as table columns
		paperStr := strings.Join(paperRarities, ", ")
		mtgoStr := strings.Join(mtgoRarities, ", ")
		arenaStr := strings.Join(arenaRarities, ", ")

		fmt.Printf("%-30s %-20s %-20s %-20s\n", cardName, paperStr, mtgoStr, arenaStr)
	}

	return nil
}

func run() error {
	ctx := context.Background()

	// Check if database file exists
	dbFile := "scryfall.db"
	_, err := os.Stat(dbFile)
	dbExists := err == nil

	db, err := sql.Open("sqlite", dbFile)
	if err != nil {
		return err
	}
	defer db.Close()

	// Create tables only if database doesn't exist
	if !dbExists {
		fmt.Println("Creating database tables...")
		if _, err := db.ExecContext(ctx, ddl); err != nil {
			return err
		}

		// If database is new, populate it with cards
		fmt.Println("Database is new, fetching and inserting cards...")
		return queryAndInsertCards(db)
	}

	// Database exists, show menu
	fmt.Println("Database exists. Choose an option:")
	fmt.Println("1. Query and insert new cards")
	fmt.Println("2. Load and display cards from database")
	fmt.Print("Enter choice (1 or 2): ")

	var choice string
	fmt.Scanln(&choice)

	switch choice {
	case "1":
		return queryAndInsertCards(db)
	case "2":
		return loadAndDisplayCards(db)
	default:
		fmt.Println("Invalid choice. Defaulting to display cards.")
		return loadAndDisplayCards(db)
	}
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

type CardPrintings struct {
	Name      string
	Printings []string
}

func getRarityAbbrev(rarity string) string {
	switch rarity {
	case "common":
		return "c"
	case "uncommon":
		return "u"
	case "rare":
		return "r"
	case "mythic":
		return "m"
	default:
		return "?"
	}
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

func oldMain() {
	client, err := NewClient("TestApp")
	if err != nil {
		panic(err)
	}

	searchQuery := "(game:paper game:mtgo -game:arena in:common or in:uncommon) game:arena r>=rare"

	fmt.Printf("Searching for query %s\n", searchQuery)
	results, err := client.SearchCards(searchQuery)
	if err != nil {
		log.Printf("Search error: %v", err)
		return
	}

	fmt.Printf("Found %d cards\n", results.TotalCards)

	var cardPrintings []CardPrintings

	for _, card := range results.Data {
		fmt.Printf("Fetching printings for %s...\n", card.Name)

		printings, err := client.GetCardPrintings(card.PrintsSearchURI.String())
		if err != nil {
			log.Printf("Error fetching printings for %s: %v", card.Name, err)
			continue
		}

		// Filter out cards that have common/uncommon Arena printings
		if !shouldIncludeCard(printings.Data) {
			fmt.Printf("Skipping %s - has common/uncommon Arena printing\n", card.Name)
			continue
		}

		var printingStrings []string
		for _, printing := range printings.Data {
			rarityAbbrev := getRarityAbbrev(printing.Rarity)
			printingStrings = append(printingStrings, fmt.Sprintf("%s %s", printing.Set, rarityAbbrev))
		}

		cardPrintings = append(cardPrintings, CardPrintings{
			Name:      card.Name,
			Printings: printingStrings,
		})

		// Rate limiting - 50-100ms delay between requests
		time.Sleep(75 * time.Millisecond)
	}

	fmt.Printf("\nFiltered to %d cards that don't have common/uncommon Arena printings\n\n", len(cardPrintings))

	// Display in table format
	displayTable(cardPrintings)
}

func displayTable(cardPrintings []CardPrintings) {
	const cardsPerRow = 3
	const columnWidth = 25

	for i := 0; i < len(cardPrintings); i += cardsPerRow {
		end := i + cardsPerRow
		if end > len(cardPrintings) {
			end = len(cardPrintings)
		}

		// Print top border
		for j := 0; j < end-i; j++ {
			fmt.Print("┌")
			fmt.Print(strings.Repeat("─", columnWidth-1))
			if j < end-i-1 {
				fmt.Print("┬")
			} else {
				fmt.Print("┐")
			}
		}
		fmt.Println()

		// Print card names
		for j := i; j < end; j++ {
			name := cardPrintings[j].Name
			if len(name) > columnWidth-3 {
				name = name[:columnWidth-6] + "..."
			}
			fmt.Printf("│ %-*s", columnWidth-2, name)
		}
		fmt.Println("│")

		// Print separator
		for j := 0; j < end-i; j++ {
			fmt.Print("├")
			fmt.Print(strings.Repeat("─", columnWidth-1))
			if j < end-i-1 {
				fmt.Print("┼")
			} else {
				fmt.Print("┤")
			}
		}
		fmt.Println()

		// Find max number of printings in this row
		maxPrintings := 0
		for j := i; j < end; j++ {
			if len(cardPrintings[j].Printings) > maxPrintings {
				maxPrintings = len(cardPrintings[j].Printings)
			}
		}

		// Print printings
		for printingRow := 0; printingRow < maxPrintings; printingRow++ {
			for j := i; j < end; j++ {
				var printing string
				if printingRow < len(cardPrintings[j].Printings) {
					printing = cardPrintings[j].Printings[printingRow]
				}
				if len(printing) > columnWidth-3 {
					printing = printing[:columnWidth-6] + "..."
				}
				fmt.Printf("│ %-*s", columnWidth-2, printing)
			}
			fmt.Println("│")
		}

		// Print bottom border
		for j := 0; j < end-i; j++ {
			fmt.Print("└")
			fmt.Print(strings.Repeat("─", columnWidth-1))
			if j < end-i-1 {
				fmt.Print("┴")
			} else {
				fmt.Print("┘")
			}
		}
		fmt.Println()
		fmt.Println() // Extra space between rows
	}
}

func examples(client *Client) {
	// Example 1: General text search (finds partial matches)
	fmt.Println("=== General Search: 'lightning' ===")
	results, err := client.SearchCards("lightning")
	if err != nil {
		log.Printf("Search error: %v", err)
	} else {
		fmt.Printf("Found %d cards\n", results.TotalCards)
		for i, card := range results.Data {
			if i >= 3 { // Show first 3 results
				break
			}
			fmt.Printf("- %s\n", card.Name)
		}
	}

	// Example 2: Exact name search using ! operator
	fmt.Println("\n=== Exact Name Search: '!Lightning Bolt' ===")
	exactResults, err := client.SearchCardsByName("Lightning Bolt")
	if err != nil {
		log.Printf("Exact search error: %v", err)
	} else {
		fmt.Printf("Found %d exact matches\n", exactResults.TotalCards)
		for _, card := range exactResults.Data {
			fmt.Printf("- %s (Set: %s)\n", card.Name, card.Set)
		}
	}

	// Example 3: Advanced search queries
	fmt.Println("\n=== Advanced Search: 'c:red cmc:3' ===")
	advancedResults, err := client.SearchCards("c:red cmc:3")
	if err != nil {
		log.Printf("Advanced search error: %v", err)
	} else {
		fmt.Printf("Found %d red cards with CMC 3\n", advancedResults.TotalCards)
		for i, card := range advancedResults.Data {
			if i >= 3 { // Show first 3 results
				break
			}
			fmt.Printf("- %s\n", card.Name)
		}
	}

	// Example 4: Search by card text
	fmt.Println("\n=== Text Search: 'o:flying' ===")
	textResults, err := client.SearchCards("o:flying")
	if err != nil {
		log.Printf("Text search error: %v", err)
	} else {
		fmt.Printf("Found %d cards with 'flying'\n", textResults.TotalCards)
		for i, card := range textResults.Data {
			if i >= 3 { // Show first 3 results
				break
			}
			fmt.Printf("- %s\n", card.Name)
		}
	}
}
