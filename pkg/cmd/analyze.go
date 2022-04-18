// Copyright 2022 Guy Barden
// track.go - top level command that provides and saves the price information from the Betfair Exchange

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

// 	http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package cmd

import (
	"fmt"
	"guysports/go-football-trader/pkg/helper"
	"guysports/go-football-trader/pkg/store"

	"github.com/guysports/go-betfair-api/pkg/types"
)

type (
	Analyze struct {
		StoreFile string `help:"Path to the where the history of price data for fixtures stored in json format"`
	}

	OddsRange struct {
		Low  float32
		High float32
	}
)

const (
	BetfairCommission = 0.02
	DefaultStake      = 100
)

var (
	// Odds ranges to analyze
	oddsRangeToTest = []OddsRange{
		{
			Low:  1.2,
			High: 1.99,
		},
		{
			Low:  2.0,
			High: 2.99,
		},
		{
			Low:  3.0,
			High: 4.99,
		},
		{
			Low:  5.0,
			High: 9.99,
		},
		{
			Low:  10.0,
			High: 19.99,
		},
		{
			Low:  20.0,
			High: 29.99,
		},
	}
)

func (a *Analyze) Run(globals *types.Globals) error {
	s := store.NewStore(a.StoreFile, nil)

	// For each fixture in the store a picture of price trending is established, initially look at back prices
	// data to mine, start price, number of price changes in trend direction and against trend direction,
	// and last price delta from start
	trends := s.ExtractTrendsFromFixtures()

	// Show delta breakdown by odds range
	oddsRangeToTest := []OddsRange{
		{
			Low:  1.2,
			High: 1.99,
		},
		{
			Low:  2.0,
			High: 2.99,
		},
		{
			Low:  3.0,
			High: 4.99,
		},
		{
			Low:  5.0,
			High: 9.99,
		},
		{
			Low:  10.0,
			High: 19.99,
		},
		{
			Low:  20.0,
			High: 29.99,
		},
	}

	for _, odds := range oddsRangeToTest {
		printTrendForOddsRangeInformation(trends, odds)
	}

	return nil
}

func printTrendForOddsRangeInformation(trends []store.Trend, odds OddsRange) {
	lineBreak()
	fmt.Printf("Price analysis in the %.2f to %.2f range\n", odds.Low, odds.High)
	var cumulativeProfit, cumulativeLoss float32
	var positive, negative int
	for _, trend := range trends {
		if trend.StartPrice < odds.Low || trend.StartPrice > odds.High {
			continue
		}
		percent := trend.Delta * 100 / trend.StartPrice
		laystake, profit := calculateProfit(trend.StartPrice, trend.CurrentPrice)
		ql := DefaultStake - laystake*(1-BetfairCommission)
		fmt.Printf("%s (%s) (%d) %.2f %.2f %.2f %.2f --- %.2f%% --- £%.2f £%.2f £%.2f\n", trend.Fixture, trend.Team, trend.SampleNumber, trend.StartPrice, trend.StartLayPrice, trend.CurrentPrice, trend.Delta, percent, laystake, ql, profit)
		if trend.Delta > 0 {
			cumulativeProfit += profit
			positive++
		} else {
			cumulativeLoss += profit
			negative++
		}
	}
	lineBreak()
	fmt.Printf("Cumulative Profit %.2f (%d)\n", cumulativeProfit, positive)
	fmt.Printf("Cumulative Loss %.2f (%d)\n", cumulativeLoss, negative)
	lineBreak()
}

func calculateProfit(backodds, layodds float32) (float32, float32) {
	laystake := (DefaultStake * backodds) / (layodds - BetfairCommission)

	// Profit = stake - laystake * (1-commission)
	// Loss = stake - laystake (no commission payable)
	profit := (laystake - DefaultStake)
	if laystake > DefaultStake {
		profit = profit * (1 - BetfairCommission)
	}
	return helper.ConvertTo2DP(laystake), helper.ConvertTo2DP(profit)
}

func lineBreak() {
	fmt.Println("__________________________________________________________________________________________")
}
