package message

// RedisAdapter defines the methods required from your underlying Redis client.
type RedisAdapter interface {
	SetPopularChat(key string, value string) error
	GetPopularChat(key string) (string, error)
	IncrChatPopularity(chatID uint, increment float64) error
	GetTopPopularChats(limit int64) ([]string, error)
}

// Cache provides an abstraction over Redis operations.
type Cache interface {
	SetPopularChat(key string, value string) error
	GetPopularChat(key string) (string, error)
	IncrChatPopularity(chatID uint, increment float64) error
	GetTopPopularChats(limit int64) ([]string, error)
}

// cache is an implementation of the Cache interface using a RedisAdapter.
type cache struct {
	redis RedisAdapter
}

// NewCache creates a new Cache instance.
func NewCache(redis RedisAdapter) Cache {
	return &cache{redis: redis}
}

func (c *cache) SetPopularChat(key string, value string) error {
	return c.redis.SetPopularChat(key, value)
}

func (c *cache) GetPopularChat(key string) (string, error) {
	return c.redis.GetPopularChat(key)
}

func (c *cache) IncrChatPopularity(chatID uint, increment float64) error {
	return c.redis.IncrChatPopularity(chatID, increment)
}

func (c *cache) GetTopPopularChats(limit int64) ([]string, error) {
	return c.redis.GetTopPopularChats(limit)
}
