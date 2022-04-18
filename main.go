// Copyright [022 Guy Barden
// main.go - Command line access to the sub-commands

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

// 	http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package main

import (
	"fmt"
	"guysports/go-football-trader/pkg/cmd"
	"os"

	"github.com/alecthomas/kong"
	"github.com/guysports/go-betfair-api/pkg/types"
)

var cli struct {
	Track   cmd.Track   `cmd:"" help:"Track back and lay prices for a given league"`
	Analyze cmd.Analyze `cmd:"" help:"Analyze price trends in fixtures"`
}

func main() {
	appkey := os.Getenv("BETFAIR_APP_KEY")
	if appkey == "" {
		fmt.Printf("BETFAIR_APP_KEY is required to be set in environment")
		os.Exit(1)
	}

	ctx := kong.Parse(&cli)
	err := ctx.Run(&types.Globals{
		AppKey: appkey,
	})
	ctx.FatalIfErrorf(err)

}
