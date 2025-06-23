package hosts

import (
	"ActQABot/conf"
	"context"
	"errors"
	"github.com/golang/glog"
	"sync"
)

type Availability struct {
	hostsEnv        *conf.HostsEnvironment
	mutex           sync.Locker
	availabilityMap map[string]chan struct{}
}

var HostAvbl Availability

func NewAvailability(hostsEnv *conf.HostsEnvironment) Availability {
	return Availability{
		hostsEnv:        hostsEnv,
		mutex:           &sync.Mutex{},
		availabilityMap: make(map[string]chan struct{}),
	}
}

func (ha *Availability) WrapJobCtx(hostName string, jobContext context.Context) (func(), error) {
	host, ok := ha.hostsEnv.Hosts[hostName]
	if !ok {
		glog.Errorf("host %s not found", hostName)
		return nil, errors.New("host not found")
	}
	hostAvbl, ok := ha.availabilityMap[hostName]
	if !ok {
		{
			ha.mutex.Lock()
			defer ha.mutex.Unlock()
			hostAvbl = make(chan struct{}, host.MaxConcurrency)
			ha.availabilityMap[hostName] = hostAvbl
		}
	}
	return func() {
		hostAvbl <- struct{}{} // lock
		defer func() {
			<-hostAvbl // unlock
		}()
		select {
		case <-jobContext.Done():
			return
		}
	}, nil
}
