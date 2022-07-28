package bootstrap

import (
	"context"
	"fmt"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/config"
	"github.com/go-redis/redis/v8"
	"github.com/google/wire"
	"github.com/hertz-contrib/cors"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
	"github.com/weplanx/server/common"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

var Provides = wire.NewSet(
	UseMongoDB,
	UseDatabase,
	UseRedis,
	UseNats,
	UseJetStream,
	UseHertz,
)

// LoadStaticValues 加载静态配置
func LoadStaticValues(path string) (values *common.Values, err error) {
	if _, err = os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("静态配置不存在，请检查路径 [%s]", path)
	}
	var b []byte
	if b, err = ioutil.ReadFile(path); err != nil {
		return
	}
	if err = yaml.Unmarshal(b, &values); err != nil {
		return
	}
	return
}

// UseMongoDB 初始化 MongoDB
// 配置文档 https://www.mongodb.com/docs/drivers/go/current/
func UseMongoDB(values *common.Values) (*mongo.Client, error) {
	return mongo.Connect(
		context.TODO(),
		options.Client().ApplyURI(values.Database.Uri),
	)
}

// UseDatabase 初始化数据库
func UseDatabase(client *mongo.Client, values *common.Values) (db *mongo.Database) {
	return client.Database(values.Database.Db)
}

// UseRedis 初始化 Redis
// 配置文档 https://github.com/go-redis/redis
func UseRedis(values *common.Values) (client *redis.Client, err error) {
	opts, err := redis.ParseURL(values.Redis.Uri)
	if err != nil {
		return
	}
	client = redis.NewClient(opts)
	if err = client.Ping(context.TODO()).Err(); err != nil {
		return
	}
	return
}

// UseNats 初始化 Nats
// 配置文档 https://docs.nats.io/using-nats/developer
// SDK https://github.com/nats-io/nats.go
func UseNats(values *common.Values) (nc *nats.Conn, err error) {
	var kp nkeys.KeyPair
	if kp, err = nkeys.FromSeed([]byte(values.Nats.Nkey)); err != nil {
		return
	}
	defer kp.Wipe()
	var pub string
	if pub, err = kp.PublicKey(); err != nil {
		return
	}
	if !nkeys.IsValidPublicUserKey(pub) {
		return nil, fmt.Errorf("nkey 验证失败")
	}
	if nc, err = nats.Connect(
		strings.Join(values.Nats.Hosts, ","),
		nats.MaxReconnects(5),
		nats.ReconnectWait(2*time.Second),
		nats.ReconnectJitter(500*time.Millisecond, 2*time.Second),
		nats.Nkey(pub, func(nonce []byte) ([]byte, error) {
			sig, _ := kp.Sign(nonce)
			return sig, nil
		}),
	); err != nil {
		return
	}
	return
}

// UseJetStream 初始化流
func UseJetStream(nc *nats.Conn) (nats.JetStreamContext, error) {
	return nc.JetStream(nats.PublishAsyncMaxPending(256))
}

// UseHertz 使用 Hertz
func UseHertz(values *common.Values) (h *server.Hertz, err error) {
	opts := []config.Option{
		server.WithHostPorts(":3000"),
	}

	if os.Getenv("MODE") != "release" {
		opts = append(opts, server.WithExitWaitTime(0))
	}

	h = server.Default(opts...)

	// 全局中间件
	h.Use(cors.New(cors.Config{
		AllowOrigins:     values.Cors.AllowOrigins,
		AllowMethods:     values.Cors.AllowMethods,
		AllowHeaders:     values.Cors.AllowHeaders,
		AllowCredentials: values.Cors.AllowCredentials,
		ExposeHeaders:    values.Cors.ExposeHeaders,
		MaxAge:           time.Duration(values.Cors.MaxAge) * time.Second,
	}))

	return
}
