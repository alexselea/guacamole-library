package redis

import (
	"encoding/json"
	"fmt"

	"github.com/go-redis/redis"
)

var RedisClient *redis.Client

type redisConnection struct {
	Hostname     string
	ConnectionID string
}

func PutNewConnectionOf(user string, hostname string, connectionID string) {
	connection := redisConnection{
		Hostname:     hostname,
		ConnectionID: connectionID,
	}

	connectionJSON, err := json.Marshal(connection)
	if err != nil {
		fmt.Println(err)
	}

	err = RedisClient.Set(user, connectionJSON, 0).Err()
	if err != nil {
		fmt.Println(err)
	}
}

// GetConn returns hostname, connectionID
func GetConn(user string) (string, string) {
	//it will return zero on non existing key
	notZero, err := RedisClient.Exists(user).Result()
	if err != nil {
		fmt.Println(err)
	}
	if notZero == 0 {
		return "", ""
	}

	connStr, err := RedisClient.Get(user).Result()
	connection := redisConnection{}
	err = json.Unmarshal([]byte(connStr), &connection)

	if err != nil {
		fmt.Println(err)
	}

	return connection.Hostname, connection.ConnectionID
}

func InitRedisClient() {
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     "redis:6379",
		Password: "password",
		DB:       0,
	})
}
