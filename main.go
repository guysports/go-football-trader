package main

import (
	"fmt"
	"guysports/go-football-trader/pkg/cmd"
	"os"

	"github.com/alecthomas/kong"
	"github.com/guysports/go-betfair-api/pkg/types"
)

var cli struct {
	Track cmd.Track `cmd:"" help:"Track back and lay prices for a given league"`
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
