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
	"guysports/go-football-trader/pkg/fake"
	"io/ioutil"
	"testing"
	"time"

	"github.com/guysports/go-betfair-api/pkg/types"
	"github.com/stretchr/testify/assert"
)

var testStore map[string]map[string]FixturePrices

func setupTestSuite(tb testing.TB) func(tb testing.TB) {
	if testStore == nil {
		// Attempt to read store, return new store if one doesn't exist
		marshaledStore, _ := ioutil.ReadFile("../../resource/store.json")

		// No need to check the error, if the store data cannot be unmarshaled start a new store
		_ = json.Unmarshal(marshaledStore, &testStore)
	}

	return func(tb testing.TB) {
		return
	}
}

func Test_returnBestPrice(t *testing.T) {
	type args struct {
		availableOdds []types.Odds
		highest       bool
	}
	var testOdds = []types.Odds{
		{
			Price: 2.12,
			Size:  1321.33,
		},
		{
			Price: 1.98,
			Size:  672.21,
		},
		{
			Price: 2.16,
			Size:  1458.00,
		},
		{
			Price: 2.14,
			Size:  253.98,
		},
	}
	tests := []struct {
		name       string
		args       args
		wantPrice  float32
		wantAmount float32
	}{
		{
			name: "return best back price",
			args: args{
				availableOdds: testOdds,
				highest:       true,
			},
			wantPrice:  2.16,
			wantAmount: 1458.00,
		},
		{
			name: "return best lay price",
			args: args{
				availableOdds: testOdds,
			},
			wantPrice:  1.98,
			wantAmount: 672.21,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPrice, gotAmount := returnBestPrice(tt.args.availableOdds, tt.args.highest)
			assert.Equal(t, tt.wantAmount, gotAmount)
			assert.Equal(t, tt.wantPrice, gotPrice)
		})
	}
}

func Test_getPriceFromRunner(t *testing.T) {
	type args struct {
		runner *types.Runner
	}
	tests := []struct {
		name string
		args args
		want *Price
	}{
		{
			name: "golden path get back and lay prices",
			args: args{
				runner: &types.Runner{
					Exchange: types.ExchangePrices{
						AvailableToBack: []types.Odds{
							{
								Price: 2.12,
								Size:  496.23,
							},
							{
								Price: 2.14,
								Size:  679.25,
							},
							{
								Price: 2.16,
								Size:  1258.93,
							},
						},
						AvailableToLay: []types.Odds{
							{
								Price: 2.08,
								Size:  87.36,
							},
							{
								Price: 2.06,
								Size:  378.78,
							},
							{
								Price: 2.04,
								Size:  862.47,
							},
						},
					},
				},
			},
			want: &Price{
				BackPrice:  2.16,
				BackAmount: 1258.93,
				LayPrice:   2.04,
				LayAmount:  862.47,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getPriceFromRunner(tt.args.runner)
			// Inject timestamp from got into want for assert
			tt.want.Timestamp = got.Timestamp
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestStore_findEventFromMarketId(t *testing.T) {
	type fields struct {
		GlobalPriceStore map[string]map[string]FixturePrices
	}
	type args struct {
		leagueId string
		marketId string
	}

	teardownSuite := setupTestSuite(t)
	defer teardownSuite(t)

	tests := []struct {
		name        string
		fields      fields
		args        args
		wantEventId string
		wantErr     bool
	}{
		{
			name: "golden path finding existing event id",
			fields: fields{
				GlobalPriceStore: testStore,
			},
			args: args{
				leagueId: "league1",
				marketId: "1.196124803",
			},
			wantEventId: "fixture2",
		},
		{
			name: "error if event league does not exist",
			fields: fields{
				GlobalPriceStore: testStore,
			},
			args: args{
				leagueId: "noleague",
				marketId: "1.196124803",
			},
			wantErr: true,
		},
		{
			name: "error if event does not exist in league",
			fields: fields{
				GlobalPriceStore: testStore,
			},
			args: args{
				leagueId: "league1",
				marketId: "nomarketid",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Store{
				GlobalPriceStore: tt.fields.GlobalPriceStore,
			}
			gotEventId, err := s.findEventFromMarketId(tt.args.leagueId, tt.args.marketId)
			if tt.wantErr {
				assert.NotNil(t, err)
			} else {
				assert.Equal(t, tt.wantEventId, gotEventId)
				assert.Nil(t, err)
			}
		})
	}
}

func TestStore_findEventFromTeams(t *testing.T) {
	type fields struct {
		GlobalPriceStore map[string]map[string]FixturePrices
	}
	type args struct {
		homeTeam string
		awayTeam string
	}
	teardownSuite := setupTestSuite(t)
	defer teardownSuite(t)
	tests := []struct {
		name         string
		fields       fields
		args         args
		wantLeagueId string
		wantEventId  string
		wantErr      bool
	}{
		{
			name: "golden path finding existing teams",
			fields: fields{
				GlobalPriceStore: testStore,
			},
			args: args{
				homeTeam: "Leeds",
				awayTeam: "Southampton",
			},
			wantLeagueId: "league1",
			wantEventId:  "fixture1",
		},
		{
			name: "error finding event from teams",
			fields: fields{
				GlobalPriceStore: testStore,
			},
			args: args{
				homeTeam: "Brighton",
				awayTeam: "Southampton",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Store{
				GlobalPriceStore: tt.fields.GlobalPriceStore,
			}
			gotLeagueId, gotEventId, err := s.findEventFromTeams(tt.args.homeTeam, tt.args.awayTeam)
			if tt.wantErr {
				assert.NotNil(t, err)
			} else {
				assert.Equal(t, tt.wantEventId, gotEventId)
				assert.Equal(t, tt.wantLeagueId, gotLeagueId)
				assert.Nil(t, err)
			}
		})
	}
}

func TestStore_AddLeaguePricesToStore(t *testing.T) {
	type fields struct {
		GlobalPriceStore               map[string]map[string]FixturePrices
		InjectListEventsError          bool
		InjectListMarketBookError      bool
		InjectListMarketCatalogueError bool
		AppendPrices                   bool
	}
	type args struct {
		queryParameters *access.MarketQuery
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		wantErr   bool
		wantStore map[string]map[string]FixturePrices
	}{
		{
			name: "add new fixture and price histories",
			fields: fields{
				GlobalPriceStore: map[string]map[string]FixturePrices{},
				AppendPrices:     true,
			},
			args: args{
				queryParameters: &access.MarketQuery{
					LeagueIds:         []string{"league1"},
					MinDaysToFixtures: 1,
					MaxDaysToFixtures: 7,
				},
			},
			wantStore: map[string]map[string]FixturePrices{
				"league1": {
					"fixture1": FixturePrices{
						Fixture:      "Mainz v Dortmund",
						Date:         "2022-04-06T16:30:00.000Z",
						MatchStatus:  "scheduled",
						EventID:      "fixture1",
						MarketID:     "1.195693926",
						HomeRunnerId: 64374,
						AwayRunnerId: 44785,
						PriceHistory: map[int][]Price{
							44785: {
								{
									Timestamp:  time.Now().Format(time.RFC3339),
									BackPrice:  2.86,
									LayPrice:   2.9,
									BackAmount: 537.2,
									LayAmount:  194.9,
								},
							},
							64374: {
								{
									Timestamp:  time.Now().Format(time.RFC3339),
									BackPrice:  3.7,
									LayPrice:   3.75,
									BackAmount: 777.45,
									LayAmount:  718.45,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "error getting events",
			fields: fields{
				GlobalPriceStore:      map[string]map[string]FixturePrices{},
				InjectListEventsError: true,
			},
			args: args{
				queryParameters: &access.MarketQuery{
					LeagueIds:         []string{"league1"},
					MinDaysToFixtures: 1,
					MaxDaysToFixtures: 7,
				},
			},
			wantErr: true,
		},
		{
			name: "error getting market catalog",
			fields: fields{
				GlobalPriceStore:               map[string]map[string]FixturePrices{},
				InjectListMarketCatalogueError: true,
			},
			args: args{
				queryParameters: &access.MarketQuery{
					LeagueIds:         []string{"league1"},
					MinDaysToFixtures: 1,
					MaxDaysToFixtures: 7,
				},
			},
			wantErr: true,
		},
		{
			name: "error getting market book",
			fields: fields{
				GlobalPriceStore:          map[string]map[string]FixturePrices{},
				InjectListMarketBookError: true,
			},
			args: args{
				queryParameters: &access.MarketQuery{
					LeagueIds:         []string{"league1"},
					MinDaysToFixtures: 1,
					MaxDaysToFixtures: 7,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.FakeQuery{
				InjectListEventsError:          tt.fields.InjectListEventsError,
				InjectListMarketCatalogueError: tt.fields.InjectListMarketCatalogueError,
				InjectListMarketBookError:      tt.fields.InjectListMarketBookError,
			}

			s := &Store{
				GlobalPriceStore: tt.fields.GlobalPriceStore,
				QueryClient:      &client,
			}
			err := s.AddLeaguePricesToStore(tt.args.queryParameters)
			if tt.wantErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tt.wantStore, s.GlobalPriceStore)
			}
			if tt.fields.AppendPrices {
				client.AppendPrices = true
				prices := tt.wantStore["league1"]["fixture1"]
				prices.PriceHistory[44785] = append(prices.PriceHistory[44785], Price{
					Timestamp:  time.Now().Format(time.RFC3339),
					BackPrice:  2.84,
					LayPrice:   2.92,
					BackAmount: 537.2,
					LayAmount:  194.9,
				})
				prices.PriceHistory[64374] = append(prices.PriceHistory[64374], Price{
					Timestamp:  time.Now().Format(time.RFC3339),
					BackPrice:  3.75,
					LayPrice:   3.8,
					BackAmount: 777.45,
					LayAmount:  718.45,
				})
				err = s.AddLeaguePricesToStore(tt.args.queryParameters)
				assert.Nil(t, err)
				assert.Equal(t, tt.wantStore, s.GlobalPriceStore)
			}
		})
	}
}
