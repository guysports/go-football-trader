// Copyright 2022 Guy Barden
// store.go - loads or creates a new store of fixture match odds and price information

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

// 	http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package store

import (
	"encoding/json"
	"guysports/go-football-trader/pkg/access"
	"io/ioutil"
	"time"

	"github.com/guysports/go-betfair-api/pkg/types"
)

type (
	Result string
	Status string

	// Global store of fixtures and their price histories
	Store struct {
		// GlobalPriceStore stores a map of key fixture and value of prices in a map with a key of the leagueID
		GlobalPriceStore map[string]map[string]FixturePrices `json:"global_price_store"`
		QueryClient      access.QueryInterface
		StorePath        string
	}

	// FixturePriceStore holds the information about the fixtures and it's prices over time
	FixturePrices struct {
		Fixture      string          `json:"fixture"`
		Date         string          `json:"date"`
		MatchStatus  Status          `json:"status"`
		OutCome      Result          `json:"outcome"`
		MarketID     string          `json:"market_id"`
		PriceHistory map[int][]Price `json:"history"`
	}

	// Price holds the price information for a tick
	Price struct {
		Timestamp  string  `json:"time_stamp"`
		BackPrice  float32 `json:"back_price"`
		LayPrice   float32 `json:"lay_price"`
		BackAmount float32 `json:"back_amount"`
		LayAmount  float32 `json:"lay_amount"`
	}
)

const (
	HomeWin   = Result("home")
	AwayWin   = Result("away")
	Draw      = Result("draw")
	Played    = Status("played")
	Scheduled = Status("scheduled")
)

func NewFixturePrices(fixture string, date string, marketid string) *FixturePrices {
	return &FixturePrices{
		Fixture:  fixture,
		Date:     date,
		MarketID: marketid,
	}
}

func (f *FixturePrices) AddPrice(runner int, priceTick *Price) {
	if _, ok := f.PriceHistory[runner]; !ok {
		f.PriceHistory[runner] = []Price{
			*priceTick,
		}
	} else {
		f.PriceHistory[runner] = append(f.PriceHistory[runner], *priceTick)
	}
}

// NewStore holds the state of the fixtures in the targetted leagues and their price trends
func NewStore(path string, qc access.QueryInterface) *Store {
	store := Store{
		GlobalPriceStore: map[string]map[string]FixturePrices{},
		StorePath:        path,
		QueryClient:      qc,
	}
	// Attempt to read store, return new store if one doesn't exist
	marshaledStore, err := ioutil.ReadFile(path)
	if err != nil {
		return &store
	}
	// No need to check the error, if the store data cannot be unmarshaled start a new store
	_ = json.Unmarshal(marshaledStore, &store.GlobalPriceStore)
	return &store
}

func (s *Store) AddLeaguePricesToStore(queryParameters *access.MarketQuery) error {
	// For each league build the filter to obtain the fixtures
	currentTime := time.Now()
	// Default to looking at fixtures from time of invocation to 7 days
	afterDate := currentTime
	if queryParameters.MinDaysToFixtures > 0 {
		afterDate = currentTime.Add(time.Duration(queryParameters.MinDaysToFixtures) * time.Hour * 24)
	}
	beforeDate := afterDate.Add(168 * time.Hour)
	if queryParameters.MaxDaysToFixtures > 0 {
		beforeDate = currentTime.Add(time.Duration(queryParameters.MaxDaysToFixtures) * time.Hour * 24)
	}

	// Build the filter to retrieve fixture markets for each league
	for _, competitionId := range queryParameters.LeagueIds {
		filter := types.MarketFilter{
			EventTypeIds:   []string{"1"}, // Football
			CompetitionIds: []string{competitionId},
			MarketStartTime: &types.TimeRange{
				From: afterDate.Format(time.RFC3339),
				To:   beforeDate.Format(time.RFC3339),
			},
		}
		events, err := s.QueryClient.ListEvents(&filter)
		if err != nil {
			return err
		}

		// Find fixtures meeting odds criteria
		eventIDs := []string{}
		for _, eventW := range events {
			//fmt.Printf("Event ID %s, Name %s, Number of Markets %d\n", eventW.Event.ID, eventW.Event.Name, eventW.MarketCount)
			// Build a filter to get the market catalogues for each league
			eventIDs = append(eventIDs, eventW.Event.ID)
		}

		// Get the market catalogues
		catalogFilter := types.MarketFilter{
			EventIds:        eventIDs,
			MarketTypeCodes: []string{"MATCH_ODDS"},
		}
		markets, err := s.QueryClient.ListMarketCatalogue(&catalogFilter, len(eventIDs), []string{"RUNNER_METADATA"})
		// For each market, get the pricing information
		marketIds := []string{}
		for _, market := range markets {
			//fmt.Printf("Market ID %s: %s, Matched %.2f\n", market.MarketId, market.MarketName, market.TotalMatched)
			marketIds = append(marketIds, market.MarketId)
		}
		marketBook, err := s.QueryClient.ListMarketBook(marketIds, &types.PriceProjection{PriceData: []string{"EX_BEST_OFFERS"}}, "EXECUTABLE", "ROLLED_UP_BY_AVG_PRICE")
		if err != nil {
			return err
		}

		// With the market books retrieved, distill into back and lay prices for the store
		for _, book := range marketBook {
			for _, runner := range book.Runners {

				for i := range runner.Exchange.AvailableToBack {
					if i >= len(runner.Exchange.AvailableToLay) {
						continue
					}
					//tw.AppendRow([]interface{}{runner.Exchange.AvailableToBack[i].Price, runner.Exchange.AvailableToBack[i].Size, runner.Exchange.AvailableToLay[i].Price, runner.Exchange.AvailableToLay[i].Size})
				}
				//fmt.Println(tw.Render())
			}
		}

	}
	return nil
}
