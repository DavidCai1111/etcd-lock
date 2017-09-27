package etcdlock

import (
	"context"
	"time"

	"github.com/coreos/etcd/etcdserver/api/v3lock/v3lockpb"
	"github.com/coreos/etcd/etcdserver/etcdserverpb"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

// Exposed errors.
var (
	ErrEmptyKey = errors.New("empty key")
)

const defaultEtcdKeyPrefix = "__etcd_lock/"

// Locker is the client for acquiring distributed locks from etcd. It should be
// created from NewLocker() function.
type Locker struct {
	etcdKeyPrefix  string
	defaultTimeout time.Duration
	leaseCli       etcdserverpb.LeaseClient
	kvCli          etcdserverpb.KVClient
	lockCli        v3lockpb.LockClient
}

// LockerOptions is the options for NewLocker() function.
type LockerOptions struct {
	Address        string
	DialOptions    []grpc.DialOption
	EtcdKeyPrefix  string
	DefaultTimeout time.Duration
}

// NewLocker creates a Locker according to the given options.
func NewLocker(options LockerOptions) (*Locker, error) {
	conn, err := grpc.Dial(options.Address, options.DialOptions...)
	if err != nil {
		return nil, err
	}

	if options.EtcdKeyPrefix == "" {
		options.EtcdKeyPrefix = defaultEtcdKeyPrefix
	}

	locker := &Locker{
		etcdKeyPrefix: options.EtcdKeyPrefix,
		leaseCli:      etcdserverpb.NewLeaseClient(conn),
		kvCli:         etcdserverpb.NewKVClient(conn),
		lockCli:       v3lockpb.NewLockClient(conn),
	}

	return locker, nil
}

// Lock acquires a distributed lock for the specified resource
// from etcd v3.
func (l *Locker) Lock(ctx context.Context, keyName string, timeout ...time.Duration) (*Lock, error) {
	if keyName == "" {
		return nil, errors.WithStack(ErrEmptyKey)
	}

	var ttl time.Duration
	if len(timeout) == 0 {
		ttl = l.defaultTimeout
	} else {
		ttl = timeout[0]
	}

	lease, err := l.leaseCli.LeaseGrant(ctx, &etcdserverpb.LeaseGrantRequest{
		TTL: int64(ttl.Seconds()),
	})

	if err != nil {
		return nil, errors.WithStack(err)
	}

	lockRes, err := l.lockCli.Lock(ctx, &v3lockpb.LockRequest{
		Name:  l.assembleKeyName(keyName),
		Lease: lease.ID,
	})

	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &Lock{locker: l, keyName: lockRes.Key}, nil
}

func (l *Locker) assembleKeyName(keyName string) []byte {
	return []byte(l.etcdKeyPrefix + keyName)
}
