// Package statistics provides synchronized thread-safe statistics
// counters for templates and services.
package statistics

import (
	"sync/atomic"
	"time"
)

type ServiceSync struct {
	handledRequests       int64
	blockedRequests       int64
	forwardedRequests     int64
	receivedBytes         int64
	sentBytes             int64
	highestProcessingTime int64
	averageProcessingTime int64
	highestResponseTime   int64
	averageResponseTime   int64
}

func NewServiceSync() *ServiceSync {
	return &ServiceSync{}
}

func (s *ServiceSync) Update(
	receivedBytes, sentBytes int,
	requestBlocked bool,
	processingTime time.Duration,
	responseTime time.Duration,
) {
	handledReqs := atomic.AddInt64(&s.handledRequests, 1)
	atomic.AddInt64(&s.receivedBytes, int64(receivedBytes))

	// Average processing time
	curAvgProcessingTime := atomic.LoadInt64(&s.averageProcessingTime)
	atomic.AddInt64(
		&s.averageProcessingTime,
		(int64(processingTime)-curAvgProcessingTime)/handledReqs,
	)

	// Highest processing time
	if int64(processingTime) > atomic.LoadInt64(&s.highestProcessingTime) {
		atomic.StoreInt64(&s.highestProcessingTime, int64(processingTime))
	}

	if requestBlocked {
		atomic.AddInt64(&s.blockedRequests, 1)
		return
	}
	atomic.AddInt64(&s.sentBytes, int64(sentBytes))
	atomic.AddInt64(&s.forwardedRequests, 1)

	// Highest response time
	if int64(responseTime) > atomic.LoadInt64(&s.highestResponseTime) {
		atomic.StoreInt64(&s.highestResponseTime, int64(responseTime))
	}

	// Average response time
	curAvgResponseTime := atomic.LoadInt64(&s.averageResponseTime)
	atomic.AddInt64(
		&s.averageResponseTime,
		(int64(responseTime)-curAvgResponseTime)/handledReqs,
	)
}

func (s *ServiceSync) GetBlockedRequests() int64 {
	return atomic.LoadInt64(&s.blockedRequests)
}

func (s *ServiceSync) GetForwardedRequests() int64 {
	return atomic.LoadInt64(&s.forwardedRequests)
}

func (s *ServiceSync) GetReceivedBytes() int64 {
	return atomic.LoadInt64(&s.receivedBytes)
}

func (s *ServiceSync) GetSentBytes() int64 {
	return atomic.LoadInt64(&s.sentBytes)
}

func (s *ServiceSync) GetHighestProcessingTime() int64 {
	return atomic.LoadInt64(&s.highestProcessingTime)
}

func (s *ServiceSync) GetAverageProcessingTime() int64 {
	return atomic.LoadInt64(&s.averageProcessingTime)
}

func (s *ServiceSync) GetHighestResponseTime() int64 {
	return atomic.LoadInt64(&s.highestResponseTime)
}

func (s *ServiceSync) GetAverageResponseTime() int64 {
	return atomic.LoadInt64(&s.averageResponseTime)
}

type TemplateSync struct {
	matches               int64
	highestProcessingTime int64
	averageProcessingTime int64
	highestResponseTime   int64
	averageResponseTime   int64
	// lastMatch: Time!
}

func NewTemplateSync() *TemplateSync {
	return &TemplateSync{}
}

func (s *TemplateSync) Update(
	processingTime, responseTime time.Duration,
) {
	matched := atomic.AddInt64(&s.matches, 1)

	// Highest processing time
	if int64(processingTime) > atomic.LoadInt64(&s.highestProcessingTime) {
		atomic.StoreInt64(&s.highestProcessingTime, int64(processingTime))
	}

	// Highest response time
	if int64(responseTime) > atomic.LoadInt64(&s.highestResponseTime) {
		atomic.StoreInt64(&s.highestResponseTime, int64(responseTime))
	}

	// Average processing time
	curAvgProcessingTime := atomic.LoadInt64(&s.averageProcessingTime)
	atomic.AddInt64(
		&s.averageProcessingTime,
		(int64(processingTime)-curAvgProcessingTime)/matched,
	)

	// Average response time
	curAvgResponseTime := atomic.LoadInt64(&s.averageResponseTime)
	atomic.AddInt64(
		&s.averageResponseTime,
		(int64(responseTime)-curAvgResponseTime)/matched,
	)
}

func (t *TemplateSync) GetMatches() int64 {
	return atomic.LoadInt64(&t.matches)
}

func (t *TemplateSync) GetHighestProcessingTime() int64 {
	return atomic.LoadInt64(&t.highestProcessingTime)
}

func (t *TemplateSync) GetAverageProcessingTime() int64 {
	return atomic.LoadInt64(&t.averageProcessingTime)
}

func (t *TemplateSync) GetHighestResponseTime() int64 {
	return atomic.LoadInt64(&t.highestResponseTime)
}

func (t *TemplateSync) GetAverageResponseTime() int64 {
	return atomic.LoadInt64(&t.averageResponseTime)
}
