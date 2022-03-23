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
	"fmt"
	"guysports/go-football-trader/pkg/access"
	"io/ioutil"
	"os"
	"strings"
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
		EventID      string          `json:"event_id"`
		MarketID     string          `json:"market_id"`
		HomeRunnerId int             `json:"home_runner"`
		AwayRunnerId int             `json:"away_runner"`
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
		fixtureEvents := []string{}
		for _, fixture := range events {
			//fmt.Printf("Event ID %s, Name %s, Number of Markets %d\n", eventW.Event.ID, eventW.Event.Name, eventW.MarketCount)
			// Build a filter to get the market catalogues for each league
			fixtureEvents = append(fixtureEvents, fixture.Event.ID)

			// Create an entry in the global price store for the fixture if it doesn't exist
			if _, ok := s.GlobalPriceStore[competitionId]; !ok {
				// Create a league entry
				s.GlobalPriceStore[competitionId] = map[string]FixturePrices{}
			}
			league := s.GlobalPriceStore[competitionId]
			if _, ok := league[fixture.Event.ID]; !ok {
				s.GlobalPriceStore[competitionId][fixture.Event.ID] = FixturePrices{
					Fixture:      fixture.Event.Name,
					Date:         fixture.Event.OpenDate,
					MatchStatus:  Scheduled,
					EventID:      fixture.Event.ID,
					PriceHistory: map[int][]Price{},
				}
			}
		}

		// Get the market catalogues
		catalogFilter := types.MarketFilter{
			EventIds:        fixtureEvents,
			MarketTypeCodes: []string{"MATCH_ODDS"},
		}
		markets, err := s.QueryClient.ListMarketCatalogue(&catalogFilter, len(fixtureEvents), []string{"RUNNER_METADATA"})
		if err != nil {
			return err
		}
		// For each market, get the pricing information
		// {
		// 	"marketId": "1.195693926",
		// 	"marketName": "Match Odds",
		// 	"totalMatched": 53933.43,
		// 	"runners": [{
		// 		"selectionId": 64374,
		// 		"runnerName": "Mainz",
		// 		"handicap": 0.0,
		// 		"sortPriority": 1,
		// 		"metadata": {
		// 			"runnerId": "64374"
		// 		}
		// 	}, {
		// 		"selectionId": 44785,
		// 		"runnerName": "Dortmund",
		// 		"handicap": 0.0,
		// 		"sortPriority": 2,
		// 		"metadata": {
		// 			"runnerId": "44785"
		// 		}
		// 	}, {
		// 		"selectionId": 58805,
		// 		"runnerName": "The Draw",
		// 		"handicap": 0.0,
		// 		"sortPriority": 3,
		// 		"metadata": {
		// 			"runnerId": "58805"
		// 		}
		// 	}]
		// }
		marketIds := []string{}
		for _, market := range markets {
			//fmt.Printf("Market ID %s: %s, Matched %.2f\n", market.MarketId, market.MarketName, market.TotalMatched)
			// Create PriceHistories keyed on runner selection IDs
			leagueId, eventId, err := s.findEventFromTeams(market.Selections[0].Name, market.Selections[1].Name)
			if err != nil {
				// cannot find fixture so continue to next one
				continue
			}
			event := s.GlobalPriceStore[leagueId][eventId]
			event.MarketID = market.MarketId
			event.HomeRunnerId = market.Selections[0].SelectionId
			event.AwayRunnerId = market.Selections[1].SelectionId
			s.GlobalPriceStore[leagueId][eventId] = event
			marketIds = append(marketIds, market.MarketId)
		}
		marketBook, err := s.QueryClient.ListMarketBook(marketIds, &types.PriceProjection{PriceData: []string{"EX_BEST_OFFERS"}}, "EXECUTABLE", "ROLLED_UP_BY_AVG_PRICE")
		if err != nil {
			return err
		}

		// {
		// 	"marketId": "1.195693926",
		// 	"isMarketDataDelayed": true,
		// 	"status": "OPEN",
		// 	"betDelay": 5,
		// 	"bspReconciled": false,
		// 	"complete": true,
		// 	"inplay": true,
		// 	"numberOfWinners": 1,
		// 	"numberOfRunners": 3,
		// 	"numberOfActiveRunners": 3,
		// 	"lastMatchTime": "2022-03-16T18:34:06.924Z",
		// 	"totalMatched": 946450.74,
		// 	"totalAvailable": 173585.43,
		// 	"crossMatching": true,
		// 	"runnersVoidable": false,
		// 	"version": 4417661231,
		// 	"runners": [{
		// 		"selectionId": 64374,
		// 		"handicap": 0.0,
		// 		"status": "ACTIVE",
		// 		"lastPriceTraded": 3.75,
		// 		"totalMatched": 0.0,
		// 		"ex": {
		// 			"availableToBack": [{
		// 				"price": 3.7,
		// 				"size": 777.45
		// 			}, {
		// 				"price": 3.65,
		// 				"size": 1164.38
		// 			}, {
		// 				"price": 3.6,
		// 				"size": 1019.41
		// 			}],
		// 			"availableToLay": [{
		// 				"price": 3.75,
		// 				"size": 718.89
		// 			}, {
		// 				"price": 3.8,
		// 				"size": 1145.15
		// 			}, {
		// 				"price": 3.85,
		// 				"size": 1505.97
		// 			}],
		// 			"tradedVolume": []
		// 		}
		// 	}, {
		// 		"selectionId": 44785,
		// 		"handicap": 0.0,
		// 		"status": "ACTIVE",
		// 		"lastPriceTraded": 2.9,
		// 		"totalMatched": 0.0,
		// 		"ex": {
		// 			"availableToBack": [{
		// 				"price": 2.88,
		// 				"size": 537.2
		// 			}, {
		// 				"price": 2.86,
		// 				"size": 167.17
		// 			}, {
		// 				"price": 2.84,
		// 				"size": 833.53
		// 			}],
		// 			"availableToLay": [{
		// 				"price": 2.9,
		// 				"size": 194.9
		// 			}, {
		// 				"price": 2.92,
		// 				"size": 1168.09
		// 			}, {
		// 				"price": 2.94,
		// 				"size": 526.12
		// 			}],
		// 			"tradedVolume": []
		// 		}
		// 	}, {
		// 		"selectionId": 58805,
		// 		"handicap": 0.0,
		// 		"status": "ACTIVE",
		// 		"lastPriceTraded": 2.6,
		// 		"totalMatched": 0.0,
		// 		"ex": {
		// 			"availableToBack": [{
		// 				"price": 2.58,
		// 				"size": 70.4
		// 			}, {
		// 				"price": 2.56,
		// 				"size": 1882.05
		// 			}, {
		// 				"price": 2.54,
		// 				"size": 2229.28
		// 			}],
		// 			"availableToLay": [{
		// 				"price": 2.6,
		// 				"size": 171.1
		// 			}, {
		// 				"price": 2.62,
		// 				"size": 904.11
		// 			}, {
		// 				"price": 2.64,
		// 				"size": 73.12
		// 			}],
		// 			"tradedVolume": []
		// 		}
		// 	}]
		// }]
		// With the market books retrieved, distill into back and lay prices for the store
		for _, book := range marketBook {
			// Find the fixture in the global store
			eventId, err := s.findEventFromMarketId(competitionId, book.MarketId)
			if err != nil {
				continue
			}
			event := s.GlobalPriceStore[competitionId][eventId]
			// Add or create the price history for back and lay
			for _, runner := range book.Runners {
				if runner.SelectionID == event.HomeRunnerId || runner.SelectionID == event.AwayRunnerId {
					// Find best back and lay prices
					price := getPriceFromRunner(&runner)
					if _, ok := event.PriceHistory[runner.SelectionID]; !ok {
						event.PriceHistory[runner.SelectionID] = []Price{
							*price,
						}
					} else {
						event.PriceHistory[runner.SelectionID] = append(event.PriceHistory[runner.SelectionID], *price)
					}
				}
			}
		}
	}
	return nil
}

func (s *Store) SaveStoreToFile() error {
	storebytes, err := json.Marshal(s.GlobalPriceStore)
	if err != nil {
		return err
	}

	now := time.Now().Unix()
	nameComponents := strings.Split(s.StorePath, ".")
	err = os.Rename(s.StorePath, fmt.Sprintf("%s_%d.%s", nameComponents[0], now, nameComponents[1]))
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(s.StorePath, storebytes, 0666)
	if err != nil {
		return err
	}
	return nil
}

func (s *Store) findEventFromTeams(homeTeam string, awayTeam string) (leagueId string, eventId string, err error) {
	// Look in each league
	for leagueId, league := range s.GlobalPriceStore {
		// Look at each fixture in the league
		for eventId, event := range league {
			// Check if the hoem and away teams are in the fixture name
			if strings.Contains(event.Fixture, homeTeam) && strings.Contains(event.Fixture, awayTeam) {
				return leagueId, eventId, nil
			}
		}
	}
	return "", "", fmt.Errorf("unable to find fixture from event Id")
}

func (s *Store) findEventFromMarketId(leagueId string, marketId string) (eventId string, err error) {
	for eventId, event := range s.GlobalPriceStore[leagueId] {
		if event.MarketID == marketId {
			return eventId, nil
		}
	}
	return "", fmt.Errorf("unable to find fixture from market Id")
}

func getPriceFromRunner(runner *types.Runner) *Price {
	now := time.Now()
	price := Price{
		Timestamp: now.Format(time.RFC3339),
	}

	price.BackPrice, price.BackAmount = returnBestPrice(runner.Exchange.AvailableToBack, true)
	price.LayPrice, price.LayAmount = returnBestPrice(runner.Exchange.AvailableToLay, true)

	return &price
}

func returnBestPrice(availableOdds []types.Odds, highest bool) (price float32, amount float32) {
	for i, odds := range availableOdds {
		if i == 0 {
			price = odds.Price
			amount = odds.Size
		} else {
			if highest {
				if odds.Price > price {
					price = odds.Price
					amount = odds.Size
				}
			} else {
				if odds.Price < price {
					price = odds.Price
					amount = odds.Size
				}
			}
		}
	}
	return price, amount
}
