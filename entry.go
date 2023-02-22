package simple_cache

type entry struct {
	key    string
	value  any
	weight uint64
}
