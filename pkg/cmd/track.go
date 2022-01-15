package cmd

import (
	"github.com/guysports/go-betfair-api/pkg/types"
)

type (
	Track struct {
		JsonLoginPath string `help:"Path to the json file containing the api login information to Betfair"`
	}
)

func (t *Track) Run(globals *types.Globals) error {

	return nil
}
