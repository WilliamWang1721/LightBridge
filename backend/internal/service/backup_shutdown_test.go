//go:build unit

package service

import (
	"github.com/WilliamWang1721/LightBridge/internal/config"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"
)

func TestBackupOperationAdmissionDoesNotRaceWithStop(t *testing.T) {
	for range 200 {
		s := NewBackupService(nil, &config.Config{}, nil, nil, nil)
		start := make(chan struct{})
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			finish, err := s.beginTrackedOperation()
			if err == nil {
				finish()
			}
		}()
		go func() { defer wg.Done(); <-start; s.Stop() }()
		close(start)
		wg.Wait()
		require.True(t, s.shuttingDown.Load())
		_, err := s.beginTrackedOperation()
		require.Error(t, err)
	}
}

func TestBackupStopWaitsForAdmittedOperation(t *testing.T) {
	s := NewBackupService(nil, &config.Config{}, nil, nil, nil)
	finish, err := s.beginTrackedOperation()
	require.NoError(t, err)
	done := make(chan struct{})
	go func() { s.Stop(); close(done) }()
	require.Eventually(t, s.shuttingDown.Load, time.Second, time.Millisecond)
	select {
	case <-done:
		t.Fatal("Stop returned early")
	default:
	}
	finish()
	require.Eventually(t, func() bool {
		select {
		case <-done:
			return true
		default:
			return false
		}
	}, time.Second, time.Millisecond)
}
