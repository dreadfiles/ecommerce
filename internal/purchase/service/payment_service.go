package service

import "context"

type PaymentService interface {
	Process(ctx context.Context, amount string) error
}
