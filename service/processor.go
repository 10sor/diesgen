package service

import (
	"diesgen/api"
	"diesgen/config"
	"diesgen/exel"
	"errors"
	log "github.com/sirupsen/logrus"
	"github.com/tealeg/xlsx"
	"os"
	"time"
)

func Process(configPath string, xlsxFile string) {
	c, err := config.GetConfig(configPath)
	if err != nil {
		return
	}

	client, err := api.GetClient(c.XToken)
	if err != nil {
		log.Error(err)
		return
	}

	j := api.GetJar(c.JarName, client.Jars)
	if j == nil {
		log.Error("jar not found")
		return
	}

	s, err := api.GetStatementForLast(c.XToken, 72*time.Hour, *j)
	if err != nil {
		log.Error(err)
		return
	}

	file, err := xlsx.OpenFile(xlsxFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Infof("xlsx file not exist, creating %s", xlsxFile)
			file = xlsx.NewFile()
		} else {
			log.Error(err)
			return
		}
	}

	err = exel.ProcessStatement(file, s, configPath)
	if err != nil {
		log.Error(err)
		return
	}

	err = file.Save(xlsxFile)
	if err != nil {
		log.Error(err)
		return
	}
}
