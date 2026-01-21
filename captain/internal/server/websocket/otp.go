package server

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

type OTP struct {
	Key      string
	Created  time.Time
	WorkerID uuid.UUID
}

type RetentionMap struct {
	data map[string]*OTP
	sync.RWMutex
}

func NewRetentionMap(context context.Context, retentionPeriod time.Duration) *RetentionMap {
	rm := RetentionMap{
		data: make(map[string]*OTP),
	}
	go rm.Retention(context, retentionPeriod)
	return &rm
}

func (rm *RetentionMap) NewOTP(workerID uuid.UUID) OTP {
	rm.Lock()
	defer rm.Unlock()
	o := OTP{
		Key:      uuid.NewString(),
		Created:  time.Now(),
		WorkerID: workerID,
	}
	(*rm).data[o.Key] = &o
	return o
}

func (rm *RetentionMap) VerifyOTP(key string) (bool, uuid.UUID) {
	rm.Lock()
	defer rm.Unlock()
	if otp, ok := (*rm).data[key]; ok {
		delete((*rm).data, key)
		return true, otp.WorkerID
	}
	return false, uuid.Nil
}

func (rm *RetentionMap) Retention(ctx context.Context, timePeriod time.Duration) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rm.Lock()
			for _, otp := range rm.data {
				if otp.Created.Add(timePeriod).Before(time.Now()) {
					delete(rm.data, otp.Key)
				}
			}
			rm.Unlock()
		case <-ctx.Done():
			return
		}
	}
}
