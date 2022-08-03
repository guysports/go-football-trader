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
	"guysports/go-football-trader/pkg/helper"
	"io/ioutil"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/guysports/go-betfair-api/pkg/types"
)

type (
	Result         string
	Status         string
	TrendDirection bool

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

	// Trend holds the information extracted from the global store to analyze for price movements
	Trend struct {
		Fixture                  string
		Team                     string
		Home                     bool
		StartTime                string
		StartPrice               float32
		StartLayPrice            float32
		CurrentPrice             float32
		CurrentLayPrice          float32
		Delta                    float32
		PriceChanges             int
		PriceChangesAgainstTrend int
		SampleNumber             int
		Trend                    TrendDirection
	}

	Trends []Trend
)

const (
	HomeWin   = Result("home")
	AwayWin   = Result("away")
	Draw      = Result("draw")
	Played    = Status("played")
	Scheduled = Status("scheduled")

	TrendingUp   = TrendDirection(true)
	TrendingDown = TrendDirection(false)
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
		// If there are no events move to the next competition
		if len(events) == 0 {
			continue
		}
		// Find fixtures meeting odds criteria
		fixtureEvents := []string{}
		for _, fixture := range events {
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
		marketIds := []string{}
		for _, market := range markets {
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
	_ = os.Rename(s.StorePath, fmt.Sprintf("%s_%d.%s", nameComponents[0], now, nameComponents[1]))
	err = ioutil.WriteFile(s.StorePath, storebytes, 0666)
	if err != nil {
		return err
	}
	return nil
}

func (s *Store) ExtractTrendsFromFixtures() (trends Trends) {
	for _, league := range s.GlobalPriceStore {
		for _, fixture := range league {
			trend := extractTrendFromFixture(fixture)
			if trend != nil {
				trends = append(trends, trend...)
			}
		}
	}
	sort.Sort(trends)

	return
}

func (t Trends) Len() int {
	return len(t)
}
func (t Trends) Less(i, j int) bool {
	return t[i].Delta < t[j].Delta
}
func (t Trends) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

func extractTrendFromFixture(fixture FixturePrices) (trend Trends) {
	teams := strings.Split(fixture.Fixture, " v ")
	if len(teams) != 2 {
		return nil
	}
	homeId := fixture.HomeRunnerId
	awayId := fixture.AwayRunnerId
	homePriceHistory := fixture.PriceHistory[homeId]
	awayPriceHistory := fixture.PriceHistory[awayId]

	// Find entry point of start & start layprice being within two ticks
	homeIdx := findStartIndexInPrices(homePriceHistory)
	if homeIdx == nil {
		return nil
	}
	homeTrend := Trend{
		Fixture:         fixture.Fixture,
		Team:            teams[0],
		Home:            true,
		StartTime:       homePriceHistory[*homeIdx].Timestamp,
		StartPrice:      homePriceHistory[*homeIdx].BackPrice,
		StartLayPrice:   homePriceHistory[*homeIdx].LayPrice,
		CurrentPrice:    homePriceHistory[len(homePriceHistory)-1].LayPrice,
		CurrentLayPrice: homePriceHistory[len(homePriceHistory)-1].BackPrice,
		Delta:           helper.ConvertTo2DP(homePriceHistory[*homeIdx].BackPrice - homePriceHistory[len(homePriceHistory)-1].LayPrice),
		SampleNumber:    len(homePriceHistory) - *homeIdx,
	}
	trend = append(trend, homeTrend)

	// Find entry point of start & start layprice being within two ticks
	awayIdx := findStartIndexInPrices(homePriceHistory)
	if awayIdx == nil {
		return nil
	}
	awayTrend := Trend{
		Fixture:         fixture.Fixture,
		Team:            teams[1],
		Home:            false,
		StartPrice:      awayPriceHistory[*awayIdx].BackPrice,
		StartLayPrice:   awayPriceHistory[*awayIdx].LayPrice,
		CurrentPrice:    awayPriceHistory[len(awayPriceHistory)-1].LayPrice,
		CurrentLayPrice: awayPriceHistory[len(awayPriceHistory)-1].BackPrice,
		Delta:           helper.ConvertTo2DP(awayPriceHistory[*awayIdx].BackPrice - awayPriceHistory[len(awayPriceHistory)-1].LayPrice),
		SampleNumber:    len(awayPriceHistory) - *awayIdx,
	}
	trend = append(trend, awayTrend)

	return trend
}

func (s *Store) findEventFromTeams(homeTeam string, awayTeam string) (leagueId string, eventId string, err error) {
	// Look in each league
	for leagueId, league := range s.GlobalPriceStore {
		// Look at each fixture in the league
		for eventId, event := range league {
			// Check if the home and away teams are in the fixture name
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
	price.LayPrice, price.LayAmount = returnBestPrice(runner.Exchange.AvailableToLay, false)

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

func findStartIndexInPrices(prices []Price) (index *int) {
	for idx, price := range prices {
		tickOffset := helper.GetBetfairTickOffset(price.BackPrice)
		if (price.LayPrice - price.BackPrice) <= 2*tickOffset {
			index = &idx
			break
		}
	}
	return index
}
