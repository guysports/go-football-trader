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

var (
	OffsetMap = map[int]float32{
		1:  0.01,
		2:  0.02,
		3:  0.05,
		4:  0.1,
		6:  0.2,
		10: 0.5,
		20: 1,
		30: 2,
	}
)

// Return a number to 2 decimal places
func ConvertTo2DP(value float32) float32 {
	return float32(math.Round(float64(value)*100) / 100)
}

// Simple table to return a betfair tick based on price
func GetBetfairTickOffset(price float32) (tick float32) {

	iprice := int(math.Floor(float64(price)))
	switch {
	case iprice <= 4:
		return OffsetMap[iprice]
	case iprice >= 4 && iprice < 6:
		return OffsetMap[4]
	case iprice >= 6 && iprice < 10:
		return OffsetMap[6]
	case iprice >= 10 && iprice < 20:
		return OffsetMap[10]
	case iprice >= 20 && iprice < 30:
		return OffsetMap[20]
	case iprice >= 30 && iprice < 50:
		return OffsetMap[30]
	}
	return 10.0
}
