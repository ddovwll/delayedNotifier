package contracts

type Retryer interface {
	Retry(fn func() error) error
}
