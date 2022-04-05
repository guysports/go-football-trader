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
	"guysports/go-football-trader/pkg/store"

	"github.com/guysports/go-betfair-api/pkg/types"
)

type (
	Analyze struct {
		StoreFile string `help:"Path to the where the history of price data for fixtures stored in json format"`
	}
)

func (a *Analyze) Run(globals *types.Globals) error {
	s := store.NewStore(a.StoreFile, nil)
	fmt.Printf("%v\n", s.GlobalPriceStore)
	return nil
}
