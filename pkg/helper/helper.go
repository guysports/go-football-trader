// Copyright 2022 Guy Barden
// helper.go - Package for utility functions

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

// 	http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package helper

import "math"

// Return a number to 2 decimal places
func ConvertTo2DP(value float32) float32 {
	return float32(math.Round(float64(value)*100) / 100)
}
