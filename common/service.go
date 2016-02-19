package common

import "sync"

type Service interface {
	Stop() error
	Start() error
}

type ServiceState int

const (
	Started ServiceState = iota
	Stopped
)

type SyncService struct {
	mutex  sync.Mutex
	wg     sync.WaitGroup
	Status ServiceState

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

	if s.Status == Stopped {
		return nil // TODO return error
	} else {
		stopRes := s.stopFn()
		if s.startBackgroundFn != nil {
			s.wg.Wait()
		}
		s.Status = Stopped
		return stopRes
	}
}

func (s *SyncService) Start() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.Status == Started {
		return nil // TODO return error
	} else {

		startRes := s.startFn()

		if s.startBackgroundFn != nil {
			s.wg.Add(1)
			go func() {
				s.startBackgroundFn()
				s.wg.Done()
			}()
		}

		s.Status = Started
		return startRes
	}
}

func (s *SyncService) WithMutex(fn func()) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	fn()
}
