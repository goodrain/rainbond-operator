package etcdutil

import (
	"github.com/coreos/etcd/clientv3"
)

// NewClient creates a new etcd client.
func NewClient(endpoints []string) (*clientv3.Client, error) {
	cfg := clientv3.Config{
		Endpoints: endpoints,
	}

	return clientv3.New(cfg)
}
