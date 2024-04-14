package datebase

import (
	"common/config"
	"common/logs"
	"context"
	"github.com/redis/go-redis/v9"
	"time"
)

type RedisManager struct {
	Cli        *redis.Client        //单机
	ClusterCli *redis.ClusterClient //集群
}

func NewRedis() *RedisManager {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var (
		cli        *redis.Client //单机
		clusterCli *redis.ClusterClient
	)
	addrs := config.Conf.Database.RedisConf.ClusterAddrs
	if len(addrs) == 0 {
		//非集群，单节点
		cli = redis.NewClient(&redis.Options{
			Addr:         config.Conf.Database.RedisConf.Addr,
			Password:     config.Conf.Database.RedisConf.Password,
			PoolSize:     config.Conf.Database.RedisConf.PoolSize,
			MinIdleConns: config.Conf.Database.RedisConf.MinIdleConns,
			DB:           0, // 默认DB 0
		})
	} else {

		clusterCli = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:        config.Conf.Database.RedisConf.ClusterAddrs,
			Password:     config.Conf.Database.RedisConf.Password,
			PoolSize:     config.Conf.Database.RedisConf.PoolSize,
			MinIdleConns: config.Conf.Database.RedisConf.MinIdleConns,
		})
	}
	//ping操作

	if clusterCli != nil {
		err := clusterCli.Ping(ctx)
		if err.Err() != nil {
			logs.Fatal("redis cluster connect err : %v", err.Err())
			return nil
		}
	}
	if cli != nil {
		err := cli.Ping(ctx)
		if err.Err() != nil {
			logs.Fatal("redis client connect err : %v", err.Err())
			return nil
		}
	}
	return &RedisManager{
		Cli:        cli,
		ClusterCli: clusterCli,
	}
}

func (r *RedisManager) Close() {
	if r.Cli != nil {
		if err := r.Cli.Close(); err != nil {
			logs.Error("redis cli close err: %v", err)
		}
	}

	if r.ClusterCli != nil {
		if err := r.ClusterCli.Close(); err != nil {
			logs.Error("redis cluster close err: %v", err)
		}
	}
}

func (r *RedisManager) Set(ctx context.Context, key string, value string, expire time.Duration) error {
	if r.Cli != nil {
		return r.Cli.Set(ctx, key, value, expire).Err()

	}

	if r.ClusterCli != nil {
		return r.ClusterCli.Set(ctx, key, value, expire).Err()
	}
	return nil
}
