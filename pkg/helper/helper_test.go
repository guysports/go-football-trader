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

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetBetfairTickOffset(t *testing.T) {
	type args struct {
		price float32
	}
	tests := []struct {
		name     string
		args     args
		wantTick float32
	}{
		{
			name: "price is between 1 and 2",
			args: args{
				price: 1.3,
			},
			wantTick: 0.01,
		},
		{
			name: "price is between 2 and 3",
			args: args{
				price: 2.99,
			},
			wantTick: 0.02,
		},
		{
			name: "price is between 3 and 4",
			args: args{
				price: 3.01,
			},
			wantTick: 0.05,
		},
		{
			name: "price is between 4 and 6",
			args: args{
				price: 5.51,
			},
			wantTick: 0.1,
		},
		{
			name: "price is between 6 and 10",
			args: args{
				price: 9.99,
			},
			wantTick: 0.2,
		},
		{
			name: "price is between 10 and 20",
			args: args{
				price: 11,
			},
			wantTick: 0.5,
		},
		{
			name: "price is between 20 and 30",
			args: args{
				price: 29.99,
			},
			wantTick: 1,
		},
		{
			name: "price is between 30 and 50",
			args: args{
				price: 30.01,
			},
			wantTick: 2,
		},
		{
			name: "price is greater than 50",
			args: args{
				price: 100,
			},
			wantTick: 10,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTick := GetBetfairTickOffset(tt.args.price)
			assert.Equal(t, tt.wantTick, gotTick)
		})
	}
}
