//go:build integration

package middleware

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestIntegration_RateLimitStrictWithRedisIgnoresForwardedFor(t *testing.T) {
	ctx := context.Background()
	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image: "redis:7-alpine", ExposedPorts: []string{"6379/tcp"},
			WaitingFor: wait.ForLog("Ready to accept connections").WithStartupTimeout(60 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("start redis: %v", err)
	}
	t.Cleanup(func() { _ = c.Terminate(context.Background()) })
	endpoint, err := c.Endpoint(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	rdb := redis.NewClient(&redis.Options{Addr: endpoint})
	defer rdb.Close()

	r := strictRouter(rdb, 3)
	var codes []int
	for i := range 5 {
		codes = append(codes, hit(r, []string{"", "1.1.1.1", "2.2.2.2", "3.3.3.3", "4.4.4.4"}[i]))
	}
	if codes[2] != http.StatusOK || codes[3] != http.StatusTooManyRequests || codes[4] != http.StatusTooManyRequests {
		t.Fatalf("codes %v: spoofed X-Forwarded-For escaped the limit", codes)
	}
	n, err := rdb.Get(ctx, "ratelimit:t:198.51.100.7").Int()
	if err != nil || n != 5 {
		t.Fatalf("redis counter for the socket address = %d %v", n, err)
	}
	if ttl := rdb.TTL(ctx, "ratelimit:t:198.51.100.7").Val(); ttl <= 0 || ttl > time.Minute {
		t.Fatalf("counter ttl %v", ttl)
	}
	// A second instance sharing Redis shares the budget.
	if code := hit(strictRouter(rdb, 3), ""); code != http.StatusTooManyRequests {
		t.Fatalf("second instance code %d", code)
	}
}
