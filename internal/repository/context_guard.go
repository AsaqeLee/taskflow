package repository

import "context"

func errIfContextDone(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}
