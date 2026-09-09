package service

import "context"

type FakePaymentService struct {
	shouldFail bool
}

func NewFakePaymentService(
	shouldFail bool,
) PaymentService {
	return &FakePaymentService{
		shouldFail: shouldFail,
	}
}

func (p *FakePaymentService) Process(
	ctx context.Context,
	amount string,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if p.shouldFail {
		return ErrPaymentDeclined
	}

	return nil
}
