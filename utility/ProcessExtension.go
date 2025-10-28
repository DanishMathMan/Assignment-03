package utility

import (
	proto "Assignment-03/grpc"
)

// LocalEvent increments the timestamp of the process and returns the new timestamp.
func LocalEvent(process *proto.Process, this chan int64) int64 {
	newTimestamp := <-this + 1
	process.Timestamp = newTimestamp
	this <- newTimestamp
	return newTimestamp
}

// RemoteEvent sets the timestamp to one greater than that of the maximum of the recipient process and the initiating
// process and returns the new timestamp.
func RemoteEvent(process *proto.Process, this chan int64, timestamp int64) int64 {
	newTimestamp := max(<-this, timestamp) + 1
	process.Timestamp = newTimestamp
	this <- newTimestamp
	return newTimestamp
}
