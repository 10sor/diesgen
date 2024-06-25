package exel

import (
	"diesgen/api"
	"diesgen/config"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/tealeg/xlsx"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

type FlatAndCard struct {
	Card string
	Flat int
}

func ProcessStatement(file *xlsx.File, statement []api.Transaction, confPath string) error {
	sheet, err := getSheet(file, sheetName())
	if err != nil {
		return err
	}

	flatIndex := getFlatToCellIndexMap(sheet)

	for _, transaction := range statement {
		pair, err := flatAndCard(transaction.Comment)

		if err != nil {
			pair, err = processFlatAndCardErr(confPath, transaction)
			if err != nil {
				return err
			}
		}

		// the case when statement contains already saved transactions
		if transactionIDExists(sheet, transaction.ID) {
			continue
		}

		updateSheet(sheet, flatIndex, transaction, pair)
	}

	return nil
}

func sheetName() string {
	now := time.Now().Local()
	name := fmt.Sprintf("%s %d", now.Month(), now.Year())
	return name
}

func updateSheet(sheet *xlsx.Sheet, flatIndex map[int]int, transaction api.Transaction, pair *FlatAndCard) {
	transactionAmount := float64(transaction.Amount / 100)

	if rowIndex, found := flatIndex[pair.Flat]; found {
		// Update existing row
		currentAmount, _ := strconv.ParseFloat(sheet.Rows[rowIndex].Cells[1].String(), 64)
		sheet.Rows[rowIndex].Cells[1].SetInt(int(math.Floor(currentAmount) + math.Floor(transactionAmount)))
		sheet.Rows[rowIndex].Cells[2].Value += "," + transaction.ID
	} else {
		// Add new row
		row := sheet.AddRow()
		row.AddCell().SetInt(pair.Flat)
		row.AddCell().SetInt(int(math.Floor(transactionAmount)))
		row.AddCell().Value = transaction.ID
		flatIndex[pair.Flat] = sheet.MaxRow - 1
	}
}

func processFlatAndCardErr(confPath string, transaction api.Transaction) (*FlatAndCard, error) {
	c, err := config.GetConfig(confPath)
	if err != nil {
		return nil, err
	}

	pair, ok := getExclusion(c.Exclusions, transaction)
	if ok {
		return pair, nil
	}

	log.Errorf("invalid comment: %s tr: %s", transaction.Comment, transaction.ID)

	e := config.Exclusion{Card: "Unknown",
		Comment:       transaction.Comment,
		TransactionID: transaction.ID,
		Amount:        transaction.Amount / 100}
	err = config.AddExclusion(confPath, e)
	if err != nil {
		return nil, err
	}

	pair.Card = e.Card
	pair.Flat = e.Flat

	return pair, nil
}

func getExclusion(exclusions []config.Exclusion, transaction api.Transaction) (*FlatAndCard, bool) {
	ok := false
	var pair FlatAndCard
	for _, exclusion := range exclusions {
		if transaction.ID == exclusion.TransactionID {
			pair.Flat = exclusion.Flat
			pair.Card = exclusion.Card
			ok = true
		}
	}
	return &pair, ok
}

func getFlatToCellIndexMap(sheet *xlsx.Sheet) map[int]int {
	flatIndex := make(map[int]int)
	for i, row := range sheet.Rows[1:] {
		if len(row.Cells) == 0 {
			// skip empty rows
			continue
		}

		flat, err := strconv.Atoi(row.Cells[0].String())
		if err != nil {
			log.Error(err)
			continue
		}
		flatIndex[flat] = i + 1
	}
	return flatIndex
}

func getSheet(file *xlsx.File, sheetName string) (*xlsx.Sheet, error) {
	const flatColumn = "Flat"
	const amountColumn = "Amount"
	const transactionsColumn = "Transactions"

	sheet := file.Sheet[sheetName]
	if sheet == nil {
		log.Infof("adding sheet: %s", sheetName)

		var err error
		sheet, err = file.AddSheet(sheetName)
		if err != nil {
			return nil, err
		}
		header := sheet.AddRow()
		header.AddCell().Value = flatColumn
		header.AddCell().Value = amountColumn
		header.AddCell().Value = transactionsColumn
	}

	return sheet, nil
}

func flatAndCard(s string) (*FlatAndCard, error) {
	// regular expression to match sequences of digits
	re := regexp.MustCompile(`\d+`)

	matches := re.FindAllString(s, -1)
	if len(matches) == 0 {
		return nil, errors.New("input does not contain flat or card")
	}

	flat, err := strconv.Atoi(matches[0])
	if err != nil {
		return nil, fmt.Errorf("invalid flat number: %w", err)
	}
	if len(strconv.Itoa(flat)) > 4 {
		return nil, fmt.Errorf("invalid flat number: %w", err)
	}

	var card string
	if len(matches) > 1 {
		card = matches[1]
	}

	return &FlatAndCard{Card: card, Flat: flat}, nil
}

func transactionIDExists(sheet *xlsx.Sheet, transactionID string) bool {
	for _, row := range sheet.Rows {
		if len(row.Cells) < 3 {
			continue
		}
		s := row.Cells[2].String()
		transactions := strings.Split(s, ",")
		if slices.Contains(transactions, transactionID) {
			return true
		}
	}
	return false
}
