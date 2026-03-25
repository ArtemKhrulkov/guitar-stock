package scraper

import (
	"github.com/sirupsen/logrus"
)

type Semaphore struct {
	permits chan struct{}
	logger  *logrus.Logger
}

func NewSemaphore(max int) *Semaphore {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

	return &Semaphore{
		permits: make(chan struct{}, max),
		logger:  logger,
	}
}

func (s *Semaphore) Acquire() {
	s.logger.Debugf("[SEMAPHORE] Acquiring permit (current: %d/%d)", len(s.permits), cap(s.permits))
	s.permits <- struct{}{}
	s.logger.Debugf("[SEMAPHORE] Acquired (current: %d/%d)", len(s.permits), cap(s.permits))
}

func (s *Semaphore) Release() {
	<-s.permits
	s.logger.Debugf("[SEMAPHORE] Released (current: %d/%d)", len(s.permits), cap(s.permits))
}

func (s *Semaphore) Current() int {
	return len(s.permits)
}

func (s *Semaphore) Max() int {
	return cap(s.permits)
}
