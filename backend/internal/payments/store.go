package payments

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/redis/go-redis/v9"
)

const paymentPrefix = "payment:"
const paymentIndex = "payments:by_created_at"

type Store interface {
	Save(context.Context, Payment) error
	List(context.Context, int64) ([]Payment, error)
	MarkEventProcessed(context.Context, string) (bool, error)
}

type RedisStore struct{ client *redis.Client }

func NewRedisStore(client *redis.Client) *RedisStore { return &RedisStore{client: client} }

func (s *RedisStore) Save(ctx context.Context, payment Payment) error {
	payload, err := json.Marshal(payment)
	if err != nil {
		return fmt.Errorf("marshal payment: %w", err)
	}
	score := float64(payment.CreatedAt.UnixMilli())
	pipe := s.client.TxPipeline()
	pipe.Set(ctx, paymentPrefix+payment.ID, payload, 0)
	pipe.ZAdd(ctx, paymentIndex, redis.Z{Score: score, Member: payment.ID})
	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("save payment: %w", err)
	}
	return nil
}

func (s *RedisStore) List(ctx context.Context, limit int64) ([]Payment, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	ids, err := s.client.ZRevRange(ctx, paymentIndex, 0, limit-1).Result()
	if err != nil {
		return nil, fmt.Errorf("list payment ids: %w", err)
	}
	if len(ids) == 0 {
		return []Payment{}, nil
	}
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = paymentPrefix + id
	}
	values, err := s.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, fmt.Errorf("load payments: %w", err)
	}
	result := make([]Payment, 0, len(values))
	for _, value := range values {
		if value == nil {
			continue
		}
		var payment Payment
		if err := json.Unmarshal([]byte(value.(string)), &payment); err != nil {
			return nil, fmt.Errorf("decode payment: %w", err)
		}
		result = append(result, payment)
	}
	sort.SliceStable(result, func(i, j int) bool { return result[i].CreatedAt.After(result[j].CreatedAt) })
	return result, nil
}

func (s *RedisStore) MarkEventProcessed(ctx context.Context, eventID string) (bool, error) {
	ok, err := s.client.SetNX(ctx, "stripe:event:"+eventID, "1", 30*24*time.Hour).Result()
	if err != nil {
		return false, fmt.Errorf("mark webhook event: %w", err)
	}
	return ok, nil
}

func Ping(ctx context.Context, client *redis.Client) error {
	if err := client.Ping(ctx).Err(); err != nil {
		return errors.New("redis unavailable: " + err.Error())
	}
	return nil
}
