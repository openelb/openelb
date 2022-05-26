// Copyright 2022 The Kubesphere Authors.

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package appctx

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

type ctxStringKey string
type ctxIntKey int

func TestGetKeyValues(t *testing.T) {
	tests := []struct {
		name string
		ctx  context.Context
		m    map[interface{}]interface{}
	}{
		{
			"Background context",
			context.Background(),
			map[interface{}]interface{}{},
		},
		{
			"Context with Values",
			context.WithValue(context.Background(), ctxStringKey("key"), "value"),
			map[interface{}]interface{}{
				ctxStringKey("key"): "value",
			},
		},
		{
			"Nested Context with Values of different types",
			context.WithValue(context.WithValue(context.Background(), ctxStringKey("key"), "value"), ctxIntKey(123), "value2"),
			map[interface{}]interface{}{
				ctxStringKey("key"): "value",
				ctxIntKey(123):      "value2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kvMap := GetKeyValuesFromCtx(tt.ctx)
			assert.Equal(t, tt.m, kvMap)
		})
	}
}

func TestPutKeyValues(t *testing.T) {
	tests := []struct {
		name string
		m    map[interface{}]interface{}
		ctx  context.Context
	}{
		{
			"empty context",
			map[interface{}]interface{}{},
			context.Background(),
		},
		{
			"single kv pair",
			map[interface{}]interface{}{
				ctxStringKey("key"): "value",
			},
			context.WithValue(context.Background(), ctxStringKey("key"), "value"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := PutKeyValuesToCtx(tt.m)
			assert.Equal(t, tt.ctx, ctx)
		})
	}
}
