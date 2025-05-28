package enrich

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	RedisClient *redis.Client
	ctx         = context.Background()
)

func InitRedis(addr string) error {
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: "",
		DB:       0,
	})

	_, err := RedisClient.Ping(ctx).Result()
	return err
}

func getCachedData(key string, apiURL string, target interface{}, expire time.Duration) error {
	val, err := RedisClient.Get(ctx, key).Bytes()
	if err == nil {
		return json.Unmarshal(val, target)
	}

	resp, err := http.Get(apiURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if err := RedisClient.Set(ctx, key, body, expire).Err(); err != nil {
		return err
	}

	return json.Unmarshal(body, target)
}

func GetAge(name string) (int, error) {
	var data struct {
		Age int `json:"age"`
	}
	err := getCachedData(
		fmt.Sprintf("age:%s", name),
		fmt.Sprintf("https://api.agify.io/?name=%s", name),
		&data,
		24*time.Hour,
	)
	return data.Age, err
}

func GetGender(name string) (string, error) {
	var data struct {
		Gender string `json:"gender"`
	}
	err := getCachedData(
		fmt.Sprintf("gender:%s", name),
		fmt.Sprintf("https://api.genderize.io/?name=%s", name),
		&data,
		24*time.Hour,
	)
	return data.Gender, err
}

func GetNationality(name string) (string, error) {
	var data struct {
		Country []struct {
			CountryID string `json:"country_id"`
		} `json:"country"`
	}
	err := getCachedData(
		fmt.Sprintf("nationality:%s", name),
		fmt.Sprintf("https://api.nationalize.io/?name=%s", name),
		&data,
		24*time.Hour,
	)
	if len(data.Country) > 0 {
		return data.Country[0].CountryID, nil
	}
	return "unknown", err
}
