package main

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	goredislib "github.com/go-redis/redis/v8"
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v8"
	"github.com/raidancampbell/gone/handlers"
	"os"
)

func initialize() (*redsync.Redsync, *goredislib.Client) {
	//  docker run -p 6379:6379 --name rds -d redis

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "127.0.0.1:6379"
	} else {
		fmt.Printf("Using overridden redis address '%s'\n", redisAddr)
	}
	client := goredislib.NewClient(&goredislib.Options{
		Addr: "redis:6379",
	})
	ping := client.Ping(context.Background())
	if ping.Err() != nil {
		panic(ping.Err())
	}
	fmt.Print("redis connection successful")
	pool := goredis.NewPool(client)

	return redsync.New(pool), client
}

func main() {
	rs, rds := initialize()
	r := gin.Default()

	r.GET("/api/v1/elect/:sid/:rid", redisWrapper(rs, rds, handlers.ElectionHandler))
	r.GET("/api/v1/elected/:sid/:rid", redisWrapper(rs, rds, handlers.ElectionQueryHandler))

	r.GET("/api/v1/complete/:sid/:rid", redisWrapper(rs, rds, handlers.CompletionHandler))
	r.GET("/api/v1/completed/:sid/:rid", redisWrapper(rs, rds, handlers.CompletionQueryHandler))

	r.GET("/alive", handlers.HealthHandler)

	addr := ":8080"
	panic(r.Run(addr))
}

func redisWrapper(rs *redsync.Redsync, rds *goredislib.Client, f func(rs *redsync.Redsync, rds *goredislib.Client, c *gin.Context)) func(c *gin.Context) {
	return func(c *gin.Context) {
		f(rs, rds, c)
	}
}
