package discovery

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type Server struct {
	Name    string `json:"name"`
	Addr    string `json:"addr"`
	Version string `json:"version"`
	Weight  int    `json:"weight"`
	Ttl     int64  `json:"ttl"`
}

// BuildRegisterKey 创建注册的key
func (s Server) BuildRegisterKey() string {
	if s.Version == "" {
		return fmt.Sprintf("/%s/%s", s.Name, s.Addr)
	}
	return fmt.Sprintf("/%s/%s/%s", s.Name, s.Version, s.Addr)
}

func ParseValue(v []byte) (Server, error) {
	var server Server
	if err := json.Unmarshal(v, &server); err != nil {
		return server, err
	}
	return server, nil
}

func ParseKey(v string) (Server, error) {
	var server Server
	//user/v1/127.0.0.1:12000或者user/127.0.0.1:12000两种格式
	strs := strings.Split(v, "/")
	if len(strs) == 2 {
		server = Server{
			Name: strs[0],
			Addr: strs[1],
		}
	} else if len(strs) == 3 {
		server = Server{
			Name:    strs[0],
			Addr:    strs[2],
			Version: strs[1],
		}
	} else {
		return Server{}, errors.New("invalid key")
	}
	return server, nil
}
