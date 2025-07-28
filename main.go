package main

import (
	"fmt"
	"log"
)

func main() {
	// Initialize client
	client, err := NewClient("MagicClubDB")
	if err != nil {
		log.Fatal(err)
	}

	// Simple menu
	fmt.Println("Choose an option:")
	fmt.Println("1. Fetch filtered cards from Scryfall API and populate database")
	fmt.Println("2. Get all filtered cards from database")
	fmt.Println("3. Search Scryfall API for cards")
	fmt.Print("Enter choice (1, 2, or 3): ")

	var choice string
	fmt.Scanln(&choice)

	switch choice {
	case "1":
		fmt.Println("Fetching filtered cards from Scryfall API...")
		if err := client.FetchFilteredScryfallAPI(); err != nil {
			log.Fatal(err)
		}
		fmt.Println("Done!")

	case "2":
		fmt.Println("Loading filtered cards from database...")
		cards, err := client.GetFilteredCards()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Found %d unique cards in database:\n", len(cards))
		for i, card := range cards {
			if i >= 10 { // Show first 10 cards
				fmt.Printf("... and %d more cards\n", len(cards)-10)
				break
			}

			// Show card info with games available
			gamesStr := "no games"
			if len(card.Games) > 0 {
				gamesStr = fmt.Sprintf("games: %v", card.Games)
			}

			manaCost := "no mana cost"
			if card.ManaCost != nil {
				manaCost = *card.ManaCost
			}

			fmt.Printf("- %s [%s] (%s) - %s\n", card.Name, manaCost, gamesStr, card.TypeLine)
		}

	case "3":
		fmt.Print("Enter search query: ")
		var query string
		fmt.Scanln(&query)

		fmt.Printf("Searching for: %s\n", query)
		cards, err := client.SearchCardsByQuery(query)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Found %d cards:\n", len(cards))
		for i, card := range cards {
			if i >= 10 { // Show first 10 results
				fmt.Printf("... and %d more cards\n", len(cards)-10)
				break
			}
			fmt.Printf("- %s (%s - %s)\n", card.Name, card.Set, card.Rarity)
		}

	default:
		fmt.Println("Invalid choice.")
	}
}
