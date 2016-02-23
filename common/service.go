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
	Stopped
	Unstarted
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
		Status:            Unstarted,
		startFn:           startFn,
		startBackgroundFn: startBackgroundFn,
		stopFn:            stopFn,
	}
}

func (s *SyncService) Stop() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.Status == Stopped || s.Status == Unstarted {
		return errors.New("Already stopped or was not started")
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
		return errors.New("Already started")
	} else {

		err := s.startFn()

		if err != nil {
			return err
		} else {
			if s.startBackgroundFn != nil {
				s.wg.Add(1)
				go func() {
					s.startBackgroundFn()
					s.wg.Done()
				}()
			}

			s.Status = Started
			return nil
		}
	}
}

func (s *SyncService) WithMutex(fn func()) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	fn()
}
