package etcdlock

// Lock is a distributed lock of specified resource which was acquired from
// etcd v3.
type Lock struct {
	locker  *Locker
	keyName []byte
}
