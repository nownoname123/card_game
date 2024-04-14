package metrics

import (
	"github.com/arl/statsviz"
	"net/http"
)

// Sever 可视化实时监控/debug/statsviz
func Sever(addr string) error {
	mux := http.NewServeMux()
	err := statsviz.Register(mux)
	if err != nil {
		return err
	}
	if err := http.ListenAndServe(addr, mux); err != nil {
		return err
	}
	return nil

}
