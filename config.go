package main

import (
	"os"
	"strings"

	"gopkg.in/ini.v1"
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

func parseINI(strPath string, objCfg *Config) error {
	objFile, err := ini.Load(strPath)
	if err != nil {
		return err
	}

	objSec := objFile.Section("")
	if v := objSec.Key("API_URL").String(); v != "" {
		objCfg.BaseURL = v
	}
	if v := objSec.Key("CLIENT_ID").String(); v != "" {
		objCfg.ClientID = v
	}
	if v := objSec.Key("CLIENT_SECRET").String(); v != "" {
		objCfg.ClientSecret = v
	}
	if v := objSec.Key("ATTACHMENTS").String(); v != "" {
		objCfg.Attachments = v
	}
	if v := objSec.Key("IN_FILE").String(); v != "" {
		objCfg.InFile = v
	}
	if v := objSec.Key("PROXY").String(); v != "" {
		objCfg.Proxy = v
	}
	if v := objSec.Key("CSV_DELIM").String(); v != "" {
		objCfg.CSVDelim = rune(v[0])
	}
	if v := objSec.Key("DEDUCTABLE").String(); v != "" {
		objCfg.Deductible = strings.ToLower(v) == "true"
	}
	if v := objSec.Key("EMPLOYEE_ID").String(); v != "" {
		objCfg.EmployeeID = v
	}
	return nil
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
