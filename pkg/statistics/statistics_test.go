package statistics_test

import (
	"testing"
	"time"

	"github.com/graph-guard/ggproxy/pkg/statistics"
	"github.com/stretchr/testify/require"
)

func TestService(t *testing.T) {
	s := statistics.NewServiceSync()

	require.Zero(t, s.GetAverageProcessingTime())
	require.Zero(t, s.GetAverageResponseTime())
	require.Zero(t, s.GetHighestProcessingTime())
	require.Zero(t, s.GetHighestResponseTime())
	require.Zero(t, s.GetBlockedRequests())
	require.Zero(t, s.GetForwardedRequests())
	require.Zero(t, s.GetReceivedBytes())
	require.Zero(t, s.GetSentBytes())

	s.Update(100, 200, false, time.Second, 2*time.Second)
	require.Equal(t, time.Second, time.Duration(s.GetAverageProcessingTime()))
	require.Equal(t, 2*time.Second, time.Duration(s.GetAverageResponseTime()))
	require.Equal(t, time.Second, time.Duration(s.GetHighestProcessingTime()))
	require.Equal(t, 2*time.Second, time.Duration(s.GetHighestResponseTime()))
	require.Equal(t, int64(0), s.GetBlockedRequests())
	require.Equal(t, int64(1), s.GetForwardedRequests())
	require.Equal(t, int64(100), s.GetReceivedBytes())
	require.Equal(t, int64(200), s.GetSentBytes())

	s.Update(100, 200, true, time.Second, time.Hour)
	require.Equal(t, time.Second, time.Duration(s.GetAverageProcessingTime()))
	require.Equal(t, 2*time.Second, time.Duration(s.GetAverageResponseTime()))
	require.Equal(t, time.Second, time.Duration(s.GetHighestProcessingTime()))
	require.Equal(t, 2*time.Second, time.Duration(s.GetHighestResponseTime()))
	require.Equal(t, int64(1), s.GetBlockedRequests())
	require.Equal(t, int64(1), s.GetForwardedRequests())
	require.Equal(t, int64(200), s.GetReceivedBytes())
	require.Equal(t, int64(200), s.GetSentBytes())

	s.Update(100, 200, false, 500*time.Millisecond, 2*time.Second)
	require.Equal(t,
		int64(833),
		time.Duration(s.GetAverageProcessingTime()).Milliseconds(),
	)
	require.Equal(t, 2*time.Second, time.Duration(s.GetAverageResponseTime()))
	require.Equal(t, time.Second, time.Duration(s.GetHighestProcessingTime()))
	require.Equal(t, 2*time.Second, time.Duration(s.GetHighestResponseTime()))
	require.Equal(t, int64(1), s.GetBlockedRequests())
	require.Equal(t, int64(2), s.GetForwardedRequests())
	require.Equal(t, int64(300), s.GetReceivedBytes())
	require.Equal(t, int64(400), s.GetSentBytes())
}

func TestTemplate(t *testing.T) {
	s := statistics.NewTemplateSync()

	require.Zero(t, s.GetMatches())
	require.Zero(t, s.GetAverageProcessingTime())
	require.Zero(t, s.GetAverageResponseTime())
	require.Zero(t, s.GetHighestProcessingTime())
	require.Zero(t, s.GetHighestResponseTime())

	s.Update(time.Second, 2*time.Second)
	require.Equal(t, int64(1), s.GetMatches())
	require.Equal(t, time.Second, time.Duration(s.GetAverageProcessingTime()))
	require.Equal(t, 2*time.Second, time.Duration(s.GetAverageResponseTime()))
	require.Equal(t, time.Second, time.Duration(s.GetHighestProcessingTime()))
	require.Equal(t, 2*time.Second, time.Duration(s.GetHighestResponseTime()))

	s.Update(time.Second, time.Second)
	require.Equal(t, int64(2), s.GetMatches())
	require.Equal(t, time.Second, time.Duration(s.GetAverageProcessingTime()))
	require.Equal(t,
		int64(1500),
		time.Duration(s.GetAverageResponseTime()).Milliseconds(),
	)
	require.Equal(t, time.Second, time.Duration(s.GetHighestProcessingTime()))
	require.Equal(t, 2*time.Second, time.Duration(s.GetHighestResponseTime()))

	s.Update(500*time.Millisecond, 2*time.Second)
	require.Equal(t, int64(3), s.GetMatches())
	require.Equal(t,
		int64(833),
		time.Duration(s.GetAverageProcessingTime()).Milliseconds(),
	)
	require.Equal(t,
		int64(1666),
		time.Duration(s.GetAverageResponseTime()).Milliseconds(),
	)
	require.Equal(t, time.Second, time.Duration(s.GetHighestProcessingTime()))
	require.Equal(t, 2*time.Second, time.Duration(s.GetHighestResponseTime()))
}
