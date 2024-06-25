package exel

import (
	"diesgen/api"
	"diesgen/config"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tealeg/xlsx"
	"path/filepath"
	"strconv"
	"testing"
)

func TestFindFlatAndCard(t *testing.T) {
	pair, err := flatAndCard("155 4441114420563932")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 155, pair.Flat)
	assert.Equal(t, "4441114420563932", pair.Card)

	pair, err = flatAndCard("кв:155 4441114420563932")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 155, pair.Flat)
	assert.Equal(t, "4441114420563932", pair.Card)

	pair, err = flatAndCard("155")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 155, pair.Flat)
	assert.Equal(t, "", pair.Card)
}

func TestProcessStatementNewSheet(t *testing.T) {
	const confName = "conf.json"
	const transactionsNumber = 10

	confDir := t.TempDir()
	confPath := filepath.Join(confDir, confName)

	var tra []api.Transaction
	for i := 0; i < transactionsNumber; i++ {
		tra = append(tra, api.Transaction{
			ID:      strconv.Itoa(i),
			Comment: fmt.Sprintf("%d %d", i, i*50),
			Amount:  i * 1000,
		})
	}

	err := config.SetConfig(confPath, config.Config{JarStart: "2024-06-25 11:00:00 +0300 EEST"})
	require.NoError(t, err)

	file := xlsx.NewFile()
	err = ProcessStatement(file, tra, confPath)
	require.NoError(t, err)

	assert.Equal(t, 1, len(file.Sheet))

	for _, sheet := range file.Sheet {
		// slice 1 for column names
		require.Equal(t, transactionsNumber, len(sheet.Rows[1:]))

		rows := sheet.Rows[1:]

		for i, row := range rows {
			flatNum, err := strconv.Atoi(row.Cells[0].Value)
			require.NoError(t, err)
			amount, err := strconv.ParseFloat(row.Cells[1].Value, 64)
			require.NoError(t, err)
			trId, err := strconv.Atoi(row.Cells[2].Value)
			require.NoError(t, err)

			assert.Equal(t, i, flatNum)
			assert.Equal(t, float64(i*10), amount)
			assert.Equal(t, i, trId)
		}
	}
	c, err := config.GetConfig(confPath)
	require.NoError(t, err)

	assert.Equal(t, 0, len(c.Exclusions))
}

func TestProcessStatementExistingSheet(t *testing.T) {
	const confName = "conf.json"
	const transactionsNumber = 10

	confDir := t.TempDir()
	confPath := filepath.Join(confDir, confName)

	var tra1 []api.Transaction
	for i := 0; i < transactionsNumber; i++ {
		tra1 = append(tra1, api.Transaction{
			ID:      strconv.Itoa(i),
			Comment: fmt.Sprintf("%d %d", i, i*50),
			Amount:  i * 1000,
		})
	}
	err := config.SetConfig(confPath, config.Config{JarStart: "2024-06-25 11:00:00 +0300 EEST"})
	if err != nil {
		t.Fatal(err)
	}
	file := xlsx.NewFile()
	err = ProcessStatement(file, tra1, confPath)
	if err != nil {
		t.Fatal(err)
	}

	// Test
	const overlapShift = 5
	var tra2 []api.Transaction
	for i := 5; i < transactionsNumber+overlapShift; i++ {
		tra2 = append(tra2, api.Transaction{
			ID:      strconv.Itoa(i * 2),
			Comment: fmt.Sprintf("%d %d", i, i*50),
			Amount:  i * 1000,
		})
	}

	err = ProcessStatement(file, tra2, confPath)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 1, len(file.Sheet))

	for _, sheet := range file.Sheet {
		// slice 1 for column names
		require.Equal(t, transactionsNumber+overlapShift, len(sheet.Rows[1:]))

		for i, row := range sheet.Rows[1:] {
			flatNum, err := strconv.Atoi(row.Cells[0].Value)
			require.NoError(t, err)
			amount, err := strconv.ParseFloat(row.Cells[1].Value, 64)
			require.NoError(t, err)

			assert.Equal(t, i, flatNum)

			if i < overlapShift {
				assert.Equal(t, float64(i*10), amount)
				assert.Equal(t, fmt.Sprintf("%d", i), row.Cells[2].Value)
				continue
			}
			if i >= transactionsNumber {
				assert.Equal(t, float64(i*10), amount)
				assert.Equal(t, fmt.Sprintf("%d", i*2), row.Cells[2].Value)
				continue
			}

			assert.Equal(t, fmt.Sprintf("%d,%d", i, i*2), row.Cells[2].Value)
			assert.Equal(t, float64(i*10)*2, amount)
		}
	}
	_ = file.Save(`C:\Users\alexm\Documents\private\logs\test\file.xlsx`)
}

func TestProcessStatementInvalidComments(t *testing.T) {
	const confName = "conf.json"

	confDir := t.TempDir()
	confPath := filepath.Join(confDir, confName)

	var tra []api.Transaction
	tra = append(tra, api.Transaction{
		ID:      "10",
		Comment: "24 4441166661984104",
		Amount:  100_000,
	})
	tra = append(tra, api.Transaction{
		ID:      "11",
		Comment: "",
		Amount:  110_000,
	})
	tra = append(tra, api.Transaction{
		ID:      "12",
		Comment: "144",
		Amount:  120_000,
	})
	tra = append(tra, api.Transaction{
		ID:      "14",
		Comment: "",
		Amount:  130_000,
	})

	err := config.SetConfig(confPath, config.Config{JarStart: "2024-06-25 11:00:00 +0300 EEST"})
	require.NoError(t, err)

	file := xlsx.NewFile()
	err = ProcessStatement(file, tra, confPath)
	require.NoError(t, err)
	err = SortMainTable(file, confPath)
	require.NoError(t, err)

	assert.Equal(t, 1, len(file.Sheet))

	for _, sheet := range file.Sheet {
		rows := sheet.Rows[1:]
		require.Equal(t, len(tra)-1, len(rows))

		firstRowCells := rows[0].Cells
		assert.Equal(t, "0", firstRowCells[flatIndex].Value)
		assert.Equal(t, "2400", firstRowCells[amountIndex].Value)
		assert.Equal(t, "11,14", firstRowCells[transactionIndex].Value)

		secondRowCells := rows[1].Cells
		assert.Equal(t, "24", secondRowCells[flatIndex].Value)
		assert.Equal(t, "1000", secondRowCells[amountIndex].Value)
		assert.Equal(t, "10", secondRowCells[transactionIndex].Value)

		thirdRowCells := rows[2].Cells
		assert.Equal(t, "144", thirdRowCells[flatIndex].Value)
		assert.Equal(t, "1200", thirdRowCells[amountIndex].Value)
		assert.Equal(t, "12", thirdRowCells[transactionIndex].Value)
	}

	c, err := config.GetConfig(confPath)
	require.NoError(t, err)

	assert.Equal(t, 2, len(c.Exclusions))
	_ = file.Save(`C:\Users\alexm\Documents\private\logs\test\text.xlsx`)
}

func TestExclusions(t *testing.T) {
	// prepare

	const confName = "conf.json"

	confDir := t.TempDir()
	confPath := filepath.Join(confDir, confName)

	var tra []api.Transaction
	tra = append(tra, api.Transaction{
		ID:      "10",
		Comment: "24 4441166661984104",
		Amount:  100_000,
	})
	tra = append(tra, api.Transaction{
		ID:      "11",
		Comment: "",
		Amount:  110_000,
	})
	tra = append(tra, api.Transaction{
		ID:      "12",
		Comment: "144",
		Amount:  120_000,
	})
	tra = append(tra, api.Transaction{
		ID:      "14",
		Comment: "",
		Amount:  130_000,
	})

	err := config.SetConfig(confPath, config.Config{JarStart: "2024-06-25 11:00:00 +0300 EEST"})
	require.NoError(t, err)

	file := xlsx.NewFile()
	err = ProcessStatement(file, tra, confPath)
	require.NoError(t, err)
	err = SortMainTable(file, confPath)
	require.NoError(t, err)

	// test

	err = config.SetConfig(
		confPath,
		config.Config{
			XToken:   "",
			JarName:  "",
			JarStart: "2024-06-25 11:00:00 +0300 EEST",
			Exclusions: []config.Exclusion{{
				Card:          "",
				Flat:          144,
				Comment:       "",
				TransactionID: "11",
				Amount:        0,
			}},
		},
	)
	require.NoError(t, err)

	err = ProcessStatement(file, tra, confPath)
	require.NoError(t, err)
	err = SortMainTable(file, confPath)
	require.NoError(t, err)

	for _, sheet := range file.Sheet {
		rows := sheet.Rows[1:]
		require.Equal(t, len(tra)-1, len(rows))

		firstRowCells := rows[0].Cells
		assert.Equal(t, "0", firstRowCells[flatIndex].Value)
		assert.Equal(t, "1300", firstRowCells[amountIndex].Value)
		assert.Equal(t, "14", firstRowCells[transactionIndex].Value)

		secondRowCells := rows[1].Cells
		assert.Equal(t, "24", secondRowCells[flatIndex].Value)
		assert.Equal(t, "1000", secondRowCells[amountIndex].Value)
		assert.Equal(t, "10", secondRowCells[transactionIndex].Value)

		thirdRowCells := rows[2].Cells
		assert.Equal(t, "144", thirdRowCells[flatIndex].Value)
		assert.Equal(t, "2300", thirdRowCells[amountIndex].Value)
		assert.Equal(t, "12,11", thirdRowCells[transactionIndex].Value)
	}

	c, err := config.GetConfig(confPath)
	require.NoError(t, err)

	assert.Equal(t, 2, len(c.Exclusions))
	_ = file.Save(`C:\Users\alexm\Documents\private\logs\test\text.xlsx`)
}
