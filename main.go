package main

import (
	"flag"
	"fmt"
	"github.com/natefinch/lumberjack"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/debug"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

const (
	serviceName   = "diesgen"
	debugFilesDir = `C:\Users\alexm\Documents\private\logs\test`
)

var (
	debugLog      = filepath.Join(debugFilesDir, "diesgen.log")
	debugXlsx     = filepath.Join(debugFilesDir, "diesgen.xlsx")
	debugConfPath = filepath.Join(debugFilesDir, "config.json")
)

func main() {
	logPath := flag.String("log", debugLog, "log file path")
	configPath := flag.String("config", debugConfPath, "config file path")
	xlsxPath := flag.String("xlsx", debugXlsx, "xlsx file path")
	flag.Parse()

	logFile := &lumberjack.Logger{
		Filename:   *logPath,
		MaxSize:    10, // Megabytes
		MaxBackups: 2,
		MaxAge:     28,   // Days
		Compress:   true, // Compress rotated files
	}

	log.SetOutput(io.MultiWriter(logFile))
	log.SetReportCaller(true)
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
		CallerPrettyfier: func(frame *runtime.Frame) (string, string) {
			funcName := filepath.Base(frame.Function)
			fileName := filepath.Base(frame.File)
			return funcName, fmt.Sprintf("%s:%d", fileName, frame.Line)
		},
	})

	log.Infof("Starting service %s", serviceName)
	log.Infof("Log path: %s", *logPath)
	log.Infof("Config path: %s", *configPath)
	log.Infof("Xlsx path: %s", *xlsxPath)

	inService, err := svc.IsWindowsService()
	if err != nil {
		log.Fatalf("failed to determine if we are running in service: %v", err)
	}

	run := svc.Run
	if !inService {
		log.SetOutput(os.Stdout)
		log.Info("Starting in debug mode")
		run = debug.Run
	} else {
		log.Info("Starting in service mode")
	}

	err = run(serviceName, NewDiesGenService(*configPath, *xlsxPath))
	if err != nil {
		log.Error("run %s service failed: %v", serviceName, err)
		return
	}
	log.Info("%s service stopped", serviceName)
}
