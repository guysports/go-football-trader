<!-- Copyright 2022 Guy Barden
   README.md provides instructions on using the tool
   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License. -->
# go-football-trader
Tracks the match odds of football leagues in Betfair and places bets to open and close positions

Usage: ./go-football-track track --help for full options

Example to track prices and save to a file based json store (created in a subdirectory `store`)
./go-football-trader track --json-login-path path-to-login-json-file --json-query path-to-query-file

Example login file
```
{
  "rootcapath": "certs/rootca.pem",
  "certpath": "client-2048.crt",
  "keypath": "client-2048.key",
  "user": "betfairUser",
  "password": "betfairPassword"
}
```

Example query file to track fixtures in Bundesliga, LaLiga, PremierLeague, Serie A for fixtures between a week and two weeks away.
(Note the odds filtering is not yet active)
```
{
    "leagueids": ["59", "81", "117", "10932509"],
    "mindays": 7,
    "maxdays": 14,
    "minodds": 2.0,
    "maxodds": 3.5
}
```
