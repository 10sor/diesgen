package service

import (
	"diesgen/api"
	"diesgen/config"
	"diesgen/exel"
	"errors"
	log "github.com/sirupsen/logrus"
	"github.com/tealeg/xlsx"
	"os"
	"slices"
)

func Process(configPath string, xlsxFile string) {
	log.Infof("START processing conf: %s, xlsx: %s", configPath, xlsxFile)

	c, err := config.GetConfig(configPath)
	if err != nil {
		log.Error(err)
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

	s, err := api.GetStatementFromToNow(c.XToken, *j, c.JarStart)
	if err != nil {
		log.Error(err)
		return
	}

	// remove withdrawals
	s = slices.DeleteFunc(s, func(transaction api.Transaction) bool {
		return transaction.Amount < 0
	})

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

	err = exel.SortMainTable(file, configPath)
	if err != nil {
		log.Error(err)
		return
	}

	err = file.Save(xlsxFile)
	if err != nil {
		log.Error(err)
		return
	}
	log.Infof("FINISH processing conf: %s, xlsx: %s", configPath, xlsxFile)
}
