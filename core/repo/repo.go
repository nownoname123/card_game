package repo

import "common/datebase"

type Manager struct {
	Mongo *datebase.MongoManager
	Redis *datebase.RedisManager
}

func New() *Manager {
	return &Manager{
		Mongo: datebase.NewMongo(),
		Redis: datebase.NewRedis(),
	}
}

func (m *Manager) Close() {
	if m.Mongo != nil {
		m.Mongo.Close()
	}
	if m.Redis != nil {
		m.Redis.Close()
	}

}
