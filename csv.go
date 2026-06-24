package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

type CSVHandler struct {
	objLogger *Logger
	chrDelim  rune
}

func NewCSVHandler(chrDelim rune, objLogger *Logger) *CSVHandler {
	return &CSVHandler{
		chrDelim:  chrDelim,
		objLogger: objLogger,
	}
}

func (c *CSVHandler) ReadCSV(strFilePath string) ([]map[string]string, error) {
	objFile, err := os.Open(strFilePath)
	if err != nil {
		return nil, fmt.Errorf("unable to open file %s: %w", strFilePath, err)
	}
	defer objFile.Close()

	objReader := csv.NewReader(objFile)
	objReader.Comma = c.chrDelim
	objReader.LazyQuotes = true

	// Read header row first
	lstHeaders, err := objReader.Read()
	if err != nil {
		return nil, fmt.Errorf("unable to read header row: %w", err)
	}

	var lstRows []map[string]string
	for {
		lstFields, err := objReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error reading CSV row: %w", err)
		}
		dictRow := make(map[string]string)
		for iIndex, strHeader := range lstHeaders {
			if iIndex < len(lstFields) {
				dictRow[strings.TrimSpace(strHeader)] = lstFields[iIndex]
			}
		}
		lstRows = append(lstRows, dictRow)
	}
	c.objLogger.LogEntry(fmt.Sprintf("Read %d rows from %s", len(lstRows), strFilePath), 1, false)
	return lstRows, nil
}

func (c *CSVHandler) WriteFailedEntries(strInFile string, lstRows []map[string]string, lstBadIDs []string) error {
	if len(lstBadIDs) == 0 {
		return nil
	}

	// Build a set of bad IDs for fast lookup
	dictBadIDs := make(map[string]bool)
	for _, strID := range lstBadIDs {
		dictBadIDs[strID] = true
	}

	// Build failed output filename
	strISO := time.Now().Format("-2006-01-02-15-04-05")
	iLoc := strings.LastIndex(strInFile, ".")
	strFailedFile := strInFile[:iLoc] + "-failed" + strISO + strInFile[iLoc:]

	objFile, err := os.Create(strFailedFile)
	if err != nil {
		return fmt.Errorf("unable to create failed entries file: %w", err)
	}
	defer objFile.Close()

	objWriter := csv.NewWriter(objFile)
	objWriter.Comma = c.chrDelim
	defer objWriter.Flush()

	// Write header
	if len(lstRows) > 0 {
		lstHeaders := make([]string, 0, len(lstRows[0]))
		for strKey := range lstRows[0] {
			lstHeaders = append(lstHeaders, strKey)
		}
		if err := objWriter.Write(lstHeaders); err != nil {
			return err
		}

		// Write matching rows
		for _, dictRow := range lstRows {
			if dictBadIDs[dictRow["Entry Number"]] {
				lstFields := make([]string, len(lstHeaders))
				for iIndex, strHeader := range lstHeaders {
					lstFields[iIndex] = dictRow[strHeader]
				}
				if err := objWriter.Write(lstFields); err != nil {
					return err
				}
				c.objLogger.LogEntry(fmt.Sprintf("Wrote bad entry ID: %s to failed file", dictRow["Entry Number"]), 1, false)
			}
		}
	}
	c.objLogger.LogEntry(fmt.Sprintf("Bad entries written to file: %s", strFailedFile), 0, false)
	return nil
}
