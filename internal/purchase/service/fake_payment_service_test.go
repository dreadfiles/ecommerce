package service

import (
	"context"
	"errors"
	"testing"
)

func TestFakePaymentService_Process(t *testing.T) {
	tests := []struct {
		name       string
		shouldFail bool
		ctx        context.Context
		wantErr    error
	}{
		{
			name:       "successful payment",
			shouldFail: false,
			ctx:        context.Background(),
			wantErr:    nil,
		},
		{
			name:       "declined payment",
			shouldFail: true,
			ctx:        context.Background(),
			wantErr:    ErrPaymentDeclined,
		},
		{
			name:       "returns context cancellation",
			shouldFail: false,
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(
					context.Background(),
				)
				cancel()
				return ctx
			}(),
			wantErr: context.Canceled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paymentService := NewFakePaymentService(
				tt.shouldFail,
			)

			err := paymentService.Process(
				tt.ctx,
				"100.00",
			)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf(
					"expected error %v, got %v",
					tt.wantErr,
					err,
				)
			}
		})
	}
}
