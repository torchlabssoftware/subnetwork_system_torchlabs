package services

import (
	"fmt"
	"log"
	"runtime/debug"

	"github.com/snail007/goproxy/manager"
)

type Service interface {
	Start(args interface{}, validator func(string, string) bool, upstreamMgr *manager.UpstreamManager, worker *manager.Worker) (err error)
	Clean()
}
type ServiceItem struct {
	S    Service
	Args interface{}
	Name string
}

var servicesMap = map[string]*ServiceItem{}

func Regist(name string, s Service, args interface{}) {
	servicesMap[name] = &ServiceItem{
		S:    s,
		Args: args,
		Name: name,
	}
}
func Run(name string, validator func(string, string) bool, upstreamMgr *manager.UpstreamManager, worker *manager.Worker) (service *ServiceItem, err error) {
	service, ok := servicesMap[name]
	if ok {
		go func() {
			defer func() {
				err := recover()
				if err != nil {
					log.Fatalf("%s servcie crashed, ERR: %s\ntrace:%s", name, err, string(debug.Stack()))
				}
			}()
			err := service.S.Start(service.Args, validator, upstreamMgr, worker)
			if err != nil {
				log.Fatalf("%s servcie fail, ERR: %s", name, err)
			}
		}()
	}
	if !ok {
		err = fmt.Errorf("service %s not found", name)
	}
	return
}
