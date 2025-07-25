package main

import (
	"fmt"
	"log"

	"github.com/kr/pretty"
)

func main() {
	client, err := NewClient("TestApp")
	if err != nil {
		panic(err)
	}

	searchQuery := "(game:paper game:mtgo -game:arena in:common or in:uncommon) game:arena r>=rare"
	if results, err := client.SearchCards(searchQuery); err != nil {
		fmt.Println(results.TotalCards)
		for _, card := range results.Data {
			pretty.Println(card)
		}
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
