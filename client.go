package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

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
	return &Client{
		baseURL:   APIBaseURL,
		userAgent: DefaultUserAgent,
		accept:    DefaultAccept,
		client:    &http.Client{},
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

func (c *Client) GetCard(id string) (*Card, error) {
	var card Card
	err := c.makeRequest("/cards/"+url.PathEscape(id), &card)
	return &card, err
}

func (c *Client) GetSet(code string) (*Set, error) {
	var set Set
	err := c.makeRequest("/sets/"+url.PathEscape(code), &set)
	return &set, err
}

func (c *Client) SearchCards(query string) (*List, error) {
	var list List
	err := c.makeRequest("/cards/search?q="+url.QueryEscape(query), &list)
	return &list, err
}

func (c *Client) SearchCardsByName(name string) (*List, error) {
	var list List
	query := "!\"" + name + "\""
	err := c.makeRequest("/cards/search?q="+url.QueryEscape(query), &list)
	return &list, err
}
