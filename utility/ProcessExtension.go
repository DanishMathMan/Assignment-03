package utility

import (
	proto "Assignment-03/grpc"
	"sync"
)

// LocalEvent increments the timestamp of the process and returns the new timestamp.
//
// this is the process
//
// lock is a mutex which is used to ensure that the timestamp incremented to is also the timestamp returned, in the event
// a separate go routine would also increment the timestamp, causing race conditions
func LocalEvent(this *proto.Process, lock *sync.Mutex) int64 {
	lock.Lock()
	this.Timestamp += 1
	defer lock.Unlock()
	return this.GetTimestamp()
}

// RemoteEvent sets the timestamp to one greater than that of the maximum of the recipient process and the initiating
// process and returns the new timestamp.
//
// this is the recipient process
//
// other is the initiating remote process
//
// lock is a mutex which is used to ensure that the timestamp incremented to is also the timestamp returned, in the event
// a separate go routine would also increment the timestamp, causing race conditions
func RemoteEvent(this *proto.Process, timestamp int64, lock *sync.Mutex) int64 {
	lock.Lock()
	this.Timestamp = max(this.GetTimestamp(), timestamp) + 1
	defer lock.Unlock()
	return this.GetTimestamp()
}
