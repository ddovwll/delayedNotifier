package retry

import "github.com/wb-go/wbf/retry"

type Retryer struct {
	strategy retry.Strategy
}

func NewRetryer(strategy retry.Strategy) *Retryer {
	return &Retryer{
		strategy: strategy,
	}
}

func (r *Retryer) Retry(fn func() error) error {
	return retry.Do(fn, r.strategy)
}
