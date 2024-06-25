package main

import (
	"diesgen/service"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/windows/svc"
	"time"
)

type DiesGenService struct {
	ConfigPath string
	XlsxPath   string
}

func NewDiesGenService(configPath string, xlsxPath string) *DiesGenService {
	return &DiesGenService{ConfigPath: configPath, XlsxPath: xlsxPath}
}

func (m *DiesGenService) Execute(_ []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (svcSpecificEC bool, errno uint32) {
	changes <- svc.Status{State: svc.StartPending}
	log.Info("StartPending")

	processTick := time.Tick(60 * time.Second)

	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
	log.Info("Running")

	service.Process(m.ConfigPath, m.XlsxPath)

loop:
	for {
		select {
		case <-processTick:
			service.Process(m.ConfigPath, m.XlsxPath)
		case c := <-r:
			switch c.Cmd {
			case svc.Stop, svc.Shutdown:
				log.Info("Stop or Shutdown received")
				break loop
			case svc.Pause:
				log.Info("Pause received")
				changes <- svc.Status{State: svc.Paused, Accepts: cmdsAccepted}
			case svc.Continue:
				log.Info("Continue received")
				changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
			default:
				log.Error("unexpected control request: %v", c)
			}
		}
	}

	changes <- svc.Status{State: svc.StopPending}
	log.Info("StopPending")
	changes <- svc.Status{State: svc.Stopped}
	log.Info("Stopped")

	return
}
