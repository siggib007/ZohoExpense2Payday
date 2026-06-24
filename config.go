package main

import (
	"bufio"
	"os"
	"strings"
)

type Config struct {
	BaseURL      string
	ClientID     string
	ClientSecret string
	Attachments  string
	InFile       string
	Proxy        string
	CSVDelim     rune
	Deductible   bool
	EmployeeID   string
	LogFile      string
	ConfFile     string
	Verbose      int
}

func defaultConfig() Config {
	return Config{
		CSVDelim:   ',',
		Deductible: true,
		EmployeeID: "name",
		Verbose:    1,
	}
}

func parseINI(path string, cfg *Config) error {
	objFile, err := os.Open(path)
	if err != nil {
		return err
	}
	defer objFile.Close()

	objScanner := bufio.NewScanner(objFile)
	for objScanner.Scan() {
		strLine := strings.TrimSpace(objScanner.Text())

		// Strip comments
		if i := strings.Index(strLine, "#"); i >= 0 {
			strLine = strings.TrimSpace(strLine[:i])
		}
		if !strings.Contains(strLine, "=") {
			continue
		}

		lstParts := strings.SplitN(strLine, "=", 2)
		strKey := strings.TrimSpace(lstParts[0])
		strVal := strings.TrimSpace(lstParts[1])

		if strVal == "" {
			continue
		}

		switch strKey {
		case "API_URL":
			cfg.BaseURL = strVal
		case "CLIENT_ID":
			cfg.ClientID = strVal
		case "CLIENT_SECRET":
			cfg.ClientSecret = strVal
		case "ATTACHMENTS":
			cfg.Attachments = strVal
		case "IN_FILE":
			cfg.InFile = strVal
		case "PROXY":
			cfg.Proxy = strVal
		case "CSV_DELIM":
			cfg.CSVDelim = rune(strVal[0])
		case "DEDUCTABLE":
			cfg.Deductible = strings.ToLower(strVal) == "true"
		case "EMPLOYEE_ID":
			cfg.EmployeeID = strVal
		}
	}
	return objScanner.Err()
}

func applyEnvVars(cfg *Config) {
	if strValue := os.Getenv("API_URL"); strValue != "" {
		cfg.BaseURL = strValue
	}
	if strValue := os.Getenv("CLIENT_ID"); strValue != "" {
		cfg.ClientID = strValue
	}
	if strValue := os.Getenv("CLIENT_SECRET"); strValue != "" {
		cfg.ClientSecret = strValue
	}
	if strValue := os.Getenv("ATTACHMENTS"); strValue != "" {
		cfg.Attachments = strValue
	}
	if strValue := os.Getenv("CSV_DELIM"); strValue != "" {
		cfg.CSVDelim = rune(strValue[0])
	}
	if strValue := os.Getenv("DEDUCTABLE"); strValue != "" {
		cfg.Deductible = strings.ToLower(strValue) == "true"
	}
	if strValue := os.Getenv("PROXY"); strValue != "" {
		cfg.Proxy = strValue
	}
	if strValue := os.Getenv("EMPLOYEE_ID"); strValue != "" {
		cfg.EmployeeID = strValue
	}
}
