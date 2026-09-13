package scheduler

import (
	"sort"
	"time"
)

func isTaskDelayed(task Task, createdAt time.Time) bool {
	if task.NotBefore.IsZero() {
		return false
	}
	if createdAt.IsZero() {
		return true
	}
	return task.NotBefore.After(createdAt)
}

func delayedWaitDurationMs(record *TaskRecord, now time.Time) int64 {
	if record == nil || now.IsZero() {
		return 0
	}
	wait := now.Sub(record.CreatedAt).Milliseconds()
	if wait < 0 {
		return 0
	}
	return wait
}

func percentileP95Int64(samples []int64) int64 {
	if len(samples) == 0 {
		return 0
	}
	cp := append([]int64(nil), samples...)
	sort.Slice(cp, func(i, j int) bool { return cp[i] < cp[j] })
	index := int(float64(len(cp))*0.95 + 0.9999999)
	if index <= 0 {
		index = 1
	}
	if index > len(cp) {
		index = len(cp)
	}
	return cp[index-1]
}
