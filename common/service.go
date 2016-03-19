package common

import (
	"errors"
	"sync"
	"sync/atomic"
)

type Service interface {
	Stop() error
	Start() error
	Status() ServiceState
}

type ServiceState int32

const (
	Started ServiceState = iota
	Stopping
	Stopped
	Unstarted
)

type SyncService struct {
	mutex  sync.Mutex
	wg     sync.WaitGroup
	status int32

	StopCh chan struct{}

	startFn           func() error
	startBackgroundFn func()
	stopFn            func() error
}

func NewSyncService(startFn func() error, startBackgroundFn func(), stopFn func() error) *SyncService {
	return &SyncService{
		status:            int32(Unstarted),
		startFn:           startFn,
		startBackgroundFn: startBackgroundFn,
		stopFn:            stopFn,
	}
}

func (s *SyncService) Stop() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	status := s.Status()

	if status == Stopped || status == Unstarted {
		return errors.New("Already stopped or not started")
	} else {
		s.status = int32(Stopping)

		close(s.StopCh)

		stopRes := s.stopFn()

		if s.startBackgroundFn != nil {
			s.wg.Wait()
		}

		s.status = int32(Stopped)
		return stopRes
	}
}

func (s *SyncService) Start() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.Status() == Started {
		return errors.New("Already started")
	} else {
		s.StopCh = make(chan struct{})

		err := s.startFn()

		if err != nil {
			return err
		} else {
			s.status = int32(Started)

			if s.startBackgroundFn != nil {
				s.wg.Add(1)
				go func() {
					s.startBackgroundFn()
					s.wg.Done()
				}()
			}

			return nil
		}
	}
}

func (s *SyncService) Status() ServiceState {
	return ServiceState(atomic.LoadInt32(&s.status))
}

func (s *SyncService) WithMutex(fn func()) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	fn()
}

func (s *SyncService) WithMutexReturning(fn func() interface{}) interface{} {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return fn()
}
