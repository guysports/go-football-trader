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
	"context"

	"guysports/go-football-trader/pkg/access"

	"guysports/go-football-trader/pkg/store"

	"github.com/guysports/go-betfair-api/pkg/types"
)

type (
	Track struct {
		JsonLoginPath string `help:"Path to the json file containing the api login information to Betfair"`
		JsonQuery     string `help:"Path to the markets to be queried for match odds"`
	}
)

func (t *Track) Run(globals *types.Globals) error {
	apiClient, err := access.NewLogin(t.JsonLoginPath)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), types.DefaultTimeout)
	defer cancel()

	// Login to Betfair
	bettingClient, err := apiClient.BetfairAuthenticate(ctx, globals.AppKey)
	if err != nil {
		return err
	}
	queryParameters, err := access.NewQuery(t.JsonQuery)
	if err != nil {
		return err
	}

	storeClient := store.NewStore("./store/store.json", bettingClient)
	err = storeClient.AddLeaguePricesToStore(queryParameters)
	if err != nil {
		return err
	}

	err = storeClient.SaveStoreToFile()
	if err != nil {
		return err
	}

	return nil
}
