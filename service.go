package raft

import "sync"

type Service interface {
	Stop() error
	Start() error
}

type ServiceState int

const (
	Started = iota
	Stopped
)

type SyncService struct {
	mutex        sync.Mutex
	wg           sync.WaitGroup
	serviceState ServiceState

	startFn           func() error
	startBackgroundFn func()
	stopFn            func() error
}

func NewSyncService(startFn func() error, startBackgroundFn func(), stopFn func() error) *SyncService {
	return &SyncService{
		startFn:           startFn,
		startBackgroundFn: startBackgroundFn,
		stopFn:            stopFn,
	}
}

func (s *SyncService) Stop() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.serviceState == Stopped {
		return nil // TODO return error
	} else {
		stopRes := s.stopFn()
		s.wg.Wait()
		s.serviceState = Stopped
		return stopRes
	}
}

func (s *SyncService) Start() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.serviceState == Started {
		return nil // TODO return error
	} else {
		s.wg.Add(1)

		startRes := s.startFn()

		go func() {
			s.startBackgroundFn()
			s.wg.Done()
		}()

		s.serviceState = Started
		return startRes
	}
}

func (s *SyncService) withMutex(fn func()) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	fn()

}
