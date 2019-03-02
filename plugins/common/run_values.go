package common

import (
	"context"

	"go.uber.org/zap"
)

// Context returns the context for the run
func (r *Run) Context() context.Context {
	if r.ctx == nil {
		r.ctx = context.Background()

		return r.ctx
	}

	return r.ctx
}

// WithContext sets the context for the run
func (r *Run) WithContext(ctx context.Context) {
	r.ctx = ctx
}

// Logger returns the logger for the run
func (r *Run) Logger() *zap.Logger {
	return r.logger
}

// WithLogger sets the logger for the run
func (r *Run) WithLogger(logger *zap.Logger) {
	r.logger = logger
}
