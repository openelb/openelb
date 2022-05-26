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

	"github.com/go-logr/logr"
)

type contextKey struct{}

// GetLogger returns a Logger constructed from ctx or nil if no
// logger details are found.
func GetLogger(ctx context.Context) logr.Logger {
	if v, ok := ctx.Value(contextKey{}).(logr.Logger); ok {
		return v
	}

	return nil
}

// WithLogger returns a new context derived from ctx that embeds the Logger.
func WithLogger(ctx context.Context, l logr.Logger) context.Context {
	return context.WithValue(ctx, contextKey{}, l)
}
