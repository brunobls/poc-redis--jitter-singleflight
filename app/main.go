package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

var (
	rdb *redis.Client
	g   singleflight.Group
)

func init() {
	host := getenv("REDIS_HOST", "localhost")
	port := getenv("REDIS_PORT", "6379")

	rdb = redis.NewClient(&redis.Options{Addr: host + ":" + port})
	rand.Seed(time.Now().UnixNano())
}

func getenv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func queryBalanceFromDB(userID string) (float64, error) {
	fmt.Printf("consultando saldo no banco para user %s...\n", userID)
	time.Sleep(200 * time.Millisecond)
	return 1234.56, nil
}

func GetUserBalance(ctx context.Context, userID string) (float64, error) {
	cacheKey := fmt.Sprintf("user:%s:balance", userID)

	if val, err := rdb.Get(ctx, cacheKey).Result(); err == nil {
		if b, err := strconv.ParseFloat(val, 64); err == nil {
			return b, nil
		}
	}

	v, err, _ := g.Do(cacheKey, func() (interface{}, error) {
		if val2, err2 := rdb.Get(ctx, cacheKey).Result(); err2 == nil {
			if b, err := strconv.ParseFloat(val2, 64); err == nil {
				return b, nil
			}
		}

		balance, err := queryBalanceFromDB(userID)
		if err != nil {
			return 0.0, err
		}

		baseTTL := 60 * time.Second
		jitter := time.Duration(rand.Int63n(int64(baseTTL / 5)))
		if rand.Intn(2) == 0 {
			jitter = -jitter
		}
		ttl := baseTTL + jitter

		fmt.Printf("definindo cache com TTL %v\n", ttl)
		if err := rdb.Set(ctx, cacheKey, fmt.Sprintf("%.2f", balance), ttl).Err(); err != nil {
			log.Printf("falha ao salvar no cache: %v", err)
		}
		return balance, nil
	})

	if err != nil {
		return 0, err
	}
	return v.(float64), nil
}

func main() {
	ctx := context.Background()
	userID := "123"

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			bal, err := GetUserBalance(ctx, userID)
			if err != nil {
				log.Printf("[goroutine %d] erro: %v", id, err)
				return
			}
			fmt.Printf("[goroutine %d] saldo = %.2f\n", id, bal)
		}(i)
	}
	wg.Wait()
}
