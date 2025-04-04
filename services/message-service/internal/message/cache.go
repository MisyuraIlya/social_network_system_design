package message

type Cache interface {
	SetPopularChat(key string, value string) error
	GetPopularChat(key string) (string, error)
}

type RedisAdapter interface {
	SetPopularChat(key string, value string) error
	GetPopularChat(key string) (string, error)
}

type cache struct {
	redis RedisAdapter
}

func NewCache(redis RedisAdapter) Cache {
	return &cache{redis: redis}
}

func (c *cache) SetPopularChat(key string, value string) error {
	return c.redis.SetPopularChat(key, value)
}

func (c *cache) GetPopularChat(key string) (string, error) {
	return c.redis.GetPopularChat(key)
}
