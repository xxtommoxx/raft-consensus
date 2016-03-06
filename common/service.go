package common

import (
	"errors"
	"sync"
)

type Service interface {
	Stop() error
	Start() error
}

type ServiceState int

const (
	Started ServiceState = iota
	Stopping
	Stopped
	Unstarted
)

type SyncService struct {
	mutex  sync.Mutex
	wg     sync.WaitGroup
	status ServiceState

	startFn           func() error
	startBackgroundFn func()
	stopFn            func() error
}

func NewSyncService(startFn func() error, startBackgroundFn func(), stopFn func() error) *SyncService {
	return &SyncService{
		status:            Unstarted,
		startFn:           startFn,
		startBackgroundFn: startBackgroundFn,
		stopFn:            stopFn,
	}
}

func (s *SyncService) Stop() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.status == Stopped || s.status == Unstarted {
		return errors.New("Already stopped or was not started")
	} else {
		s.status = Stopping
		stopRes := s.stopFn()
		if s.startBackgroundFn != nil {
			s.wg.Wait()
		}
		s.status = Stopped
		return stopRes
	}
}

func (s *SyncService) Start() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.status == Started {
		return errors.New("Already started")
	} else {

		err := s.startFn()

		if err != nil {
			return err
		} else {
			s.status = Started

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

func (s *SyncService) Status() ServiceState { // TODO use atomic read
	return s.status
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
