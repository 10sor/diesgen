package exel

import (
	"cmp"
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

const flatIndex = 0
const amountIndex = 1
const transactionIndex = 2

func ProcessStatement(file *xlsx.File, statement []api.Transaction, confPath string) error {
	sname, err := sheetName(confPath)
	if err != nil {
		return err
	}

	sheet, err := getSheet(file, sname)
	if err != nil {
		return err
	}

	flatIndexMap := getFlatToCellIndexMap(sheet)

	for _, transaction := range statement {
		transactionPair, err := flatAndCard(transaction.Comment)

		var exclusionPair *FlatAndCard
		if err != nil {
			exclusionPair, err = processFlatAndCardErr(confPath, transaction)
			if err != nil {
				return err
			}
		}

		// the case when statement contains already saved transactions
		if transactionIDExists(sheet, transaction.ID) {
			if (transactionPair == nil || transactionPair.Flat == 0) &&
				(exclusionPair != nil && exclusionPair.Flat != 0) {
				updateUnknownTransactions(sheet, transaction)
			}

			if transactionIDExists(sheet, transaction.ID) {
				continue
			}
		}

		if transactionPair == nil {
			transactionPair = exclusionPair
		}

		updateSheet(sheet, flatIndexMap, transaction, transactionPair)
	}

	return nil
}

func SortMainTable(file *xlsx.File, confPath string) error {
	sname, err := sheetName(confPath)
	if err != nil {
		return err
	}

	sheet, err := getSheet(file, sname)
	if err != nil {
		return err
	}

	slices.SortFunc(sheet.Rows, func(a, b *xlsx.Row) int {
		aFlat, err := a.Cells[flatIndex].Int()
		if err != nil {
			return -1
		}
		bFlat, err := b.Cells[flatIndex].Int()
		if err != nil {
			return 1
		}

		return cmp.Compare(aFlat, bFlat)
	})
	return nil
}

func CleanZeroAmountValues(file *xlsx.File, confPath string) error {
	sname, err := sheetName(confPath)
	if err != nil {
		return err
	}

	sheet, err := getSheet(file, sname)
	if err != nil {
		return err
	}

	if len(sheet.Rows) < 2 {
		return nil
	}

	zeroFlatRaw := sheet.Rows[1]
	if zeroFlatRaw == nil {
		return nil
	}

	if v, err := zeroFlatRaw.Cells[flatIndex].Int(); err != nil || v != 0 {
		return nil
	}

	if v, err := zeroFlatRaw.Cells[amountIndex].Int(); err != nil || v != 0 {
		return nil
	}

	sheet.Rows = append(sheet.Rows[:1], sheet.Rows[2:]...)
	return nil
}

func updateUnknownTransactions(sheet *xlsx.Sheet, tr api.Transaction) (updated bool) {
	for _, row := range sheet.Rows {
		s := row.Cells[transactionIndex].String()
		transactions := strings.Split(s, ",")
		if !slices.Contains(transactions, tr.ID) {
			continue
		}

		if flat, _ := row.Cells[flatIndex].Int(); flat != 0 {
			continue
		}

		updatedTransactions := slices.DeleteFunc(transactions, func(s string) bool {
			return s == tr.ID
		})

		row.Cells[transactionIndex].SetString(strings.Join(updatedTransactions, ","))
		amount, _ := row.Cells[amountIndex].Int()
		row.Cells[amountIndex].SetInt(amount - (tr.Amount / 100))
	}

	return false
}

func sheetName(confPath string) (string, error) {
	c, err := config.GetConfig(confPath)
	if err != nil {
		return "", err
	}

	if c.JarStart == "" {
		return "", errors.New("invalid sheet name")
	}

	layout := "2006-01-02 15:04:05 -0700 MST"
	t, err := time.Parse(layout, c.JarStart)
	if err != nil {
		return "", err
	}

	outputLayout := "2006-01-02"
	return t.Format(outputLayout), nil
}

func updateSheet(sheet *xlsx.Sheet, flatIndexMap map[int]int, transaction api.Transaction, pair *FlatAndCard) {
	transactionAmount := float64(transaction.Amount / 100)

	if rowIndex, found := flatIndexMap[pair.Flat]; found {
		// Update existing row
		currentAmount, _ := strconv.ParseFloat(sheet.Rows[rowIndex].Cells[1].String(), 64)
		sheet.Rows[rowIndex].Cells[amountIndex].SetInt(int(math.Floor(currentAmount) + math.Floor(transactionAmount)))
		sheet.Rows[rowIndex].Cells[transactionIndex].Value += "," + transaction.ID
	} else {
		// Add new row
		row := sheet.AddRow()
		row.AddCell().SetInt(pair.Flat)
		row.AddCell().SetInt(int(math.Floor(transactionAmount)))
		row.AddCell().Value = transaction.ID
		flatIndexMap[pair.Flat] = sheet.MaxRow - 1
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

	log.Errorf("invalid comment: %s tr: %s, amount: %d",
		transaction.Comment,
		transaction.ID,
		transaction.Amount/100)

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
		s := row.Cells[transactionIndex].String()
		transactions := strings.Split(s, ",")
		if slices.Contains(transactions, transactionID) {
			return true
		}
	}
	return false
}
