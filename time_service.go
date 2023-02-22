package simple_cache

type TimeService interface {
	GetCurrentTimestamp() int64
}
