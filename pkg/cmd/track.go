package cmd

import (
	"context"
	"fmt"

	"guysports/go-football-trader/pkg/access"

	"github.com/guysports/go-betfair-api/pkg/types"
)

type (
	Track struct {
		JsonLoginPath string `help:"Path to the json file containing the api login information to Betfair"`
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
	filter := types.MarketFilter{
		EventTypeIds: []string{"1", "2"},
	}
	eventTypes, err := bettingClient.ListEventTypes(&filter)
	if err != nil {
		fmt.Printf("Error returned is %s\n", err.Error())
		return err
	}
	fmt.Println("Event Types returned...")
	for _, eventTypeW := range eventTypes {
		fmt.Printf("Event Type ID %s, Market %s, Number of Markets %d\n", eventTypeW.EventType.ID, eventTypeW.EventType.Name, eventTypeW.MarketCount)
	}

	return nil
}
