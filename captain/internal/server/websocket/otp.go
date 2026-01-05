package server

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type OTP struct {
	Key      string
	Created  time.Time
	WorkerID uuid.UUID
}

type RetentionMap map[string]*OTP

func NewRetentionMap(context context.Context, retentionPeriod time.Duration) RetentionMap {
	rm := make(RetentionMap)
	go rm.Retention(context, retentionPeriod)
	return rm
}

func (rm RetentionMap) NewOTP(workerID uuid.UUID) OTP {
	o := OTP{
		Key:      uuid.NewString(),
		Created:  time.Now(),
		WorkerID: workerID,
	}
	rm[o.Key] = &o
	return o
}

func (rm RetentionMap) VerifyOTP(key string) (bool, uuid.UUID) {
	if otp, ok := rm[key]; ok {
		delete(rm, key)
		return true, otp.WorkerID
	}
	return false, uuid.Nil
}

func (rm RetentionMap) Retention(ctx context.Context, timePeriod time.Duration) {
	ticker := time.NewTicker(400 * time.Millisecond)

	for {
		select {
		case <-ticker.C:
			for _, otp := range rm {
				if otp.Created.Add(timePeriod).Before(time.Now()) {
					delete(rm, otp.Key)
				}
			}
		case <-ctx.Done():
			ticker.Stop()
			return
		}
	}
}
