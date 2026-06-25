package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	iTimeOut  = 180
	iMinQuiet = 2
)

func main() {
	// Establish base directory and script name
	strRealPath, _ := filepath.Abs(os.Args[0])
	strRealPath = strings.ReplaceAll(strRealPath, "\\", "/")
	iLoc := strings.LastIndex(strRealPath, "/")
	strBaseDir := strRealPath[:iLoc] + "/"
	strScriptName := filepath.Base(os.Args[0])
	strScriptHost, _ := os.Hostname()
	strScriptHost = strings.ToUpper(strScriptHost)
	strISO := time.Now().Format("-2006-01-02-15-04-05")

	// Log directory
	strLogDir := strBaseDir + "Logs/"
	chkdir(strLogDir)

	// Default config and log file paths
	iLoc = strings.LastIndex(strScriptName, ".")
	strDefLogFile := strLogDir + strScriptName[:iLoc] + strISO + ".log"
	strDefConf := strRealPath[:strings.LastIndex(strRealPath, ".")] + ".ini"

	// CLI flags
	strInputFile := flag.String("i", "", "Path to expense CSV file to be processed")
	bPrompt := flag.Bool("p", false, "Prompt for input file")
	strAttachments := flag.String("a", "", "Path to attachments directory")
	strDeductible := flag.String("d", "", "Is VAT deductible? True/False")
	iVerbose := flag.Int("v", 1, "Verbosity level (1-5)")
	strConfFile := flag.String("c", strDefConf, "Path to configuration file")
	strBaseURL := flag.String("u", "", "Base URL for API calls")
	strEmployee := flag.String("e", "", "Employee identification: name, kt or kennitala")
	strProxy := flag.String("x", "", "Proxy for API calls")
	strLogFile := flag.String("l", strDefLogFile, "Path to log file")
	flag.Parse()

	fmt.Printf("This is a script to transfer expense items from Zoho Expense to Payday.\n")
	fmt.Printf("Running from: %s\n", strRealPath)
	fmt.Printf("The time now is %s\n", time.Now().Format("Monday 02 January 2006 15:04:05"))
	fmt.Printf("Logs saved to %s\n", *strLogFile)

	// Initialize logger
	objLogger, err := NewLogger(*strLogFile, *iVerbose)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create log file: %s\n", err)
		os.Exit(1)
	}
	defer objLogger.Close()

	objLogger.Log(fmt.Sprintf("Starting up script %s on %s", strScriptName, strScriptHost))
	objLogger.Log(fmt.Sprintf("Verbosity set to %d", *iVerbose))
	objLogger.Log(fmt.Sprintf("Config file set to: %s", *strConfFile))

	// Load config — three tier: INI -> env vars -> CLI flags
	objCfg := defaultConfig()
	objCfg.Verbose = *iVerbose

	if err := parseINI(*strConfFile, &objCfg); err != nil {
		objLogger.Log(fmt.Sprintf("Could not read config file %s: %s", *strConfFile, err))
	}
	applyEnvVars(&objCfg)

	// CLI flags override everything
	if *strInputFile != "" {
		objCfg.InFile = *strInputFile
	}
	if *strBaseURL != "" {
		objCfg.BaseURL = *strBaseURL
	}
	if *strAttachments != "" {
		objCfg.Attachments = *strAttachments
	}
	if *strEmployee != "" {
		objCfg.EmployeeID = *strEmployee
	}
	if *strProxy != "" {
		objCfg.Proxy = *strProxy
	}
	if *strDeductible != "" {
		objCfg.Deductible = strings.ToLower(*strDeductible) == "true"
	}

	// Validate required config
	if objCfg.BaseURL == "" || objCfg.ClientID == "" || objCfg.ClientSecret == "" {
		objLogger.LogEntry("No URL or API auth config, exiting", 0, true)
	}
	if !strings.HasSuffix(objCfg.BaseURL, "/") {
		objCfg.BaseURL += "/"
	}

	// Validate employee ID type
	strEmpLower := strings.ToLower(objCfg.EmployeeID)
	if strEmpLower != "name" && strEmpLower != "kt" && strEmpLower != "kennitala" {
		objLogger.LogEntry("Employee ID must be either name, kt or kennitala", 0, true)
	}

	// Handle zip extraction or validate attachments path
	if isZipFile(objCfg.Attachments) {
		if _, err := os.Stat(objCfg.Attachments); os.IsNotExist(err) {
			objLogger.LogEntry(fmt.Sprintf("Zip file %s does not exist", objCfg.Attachments), 0, true)
		}
		strTempDir, err := extractZip(objCfg.Attachments, objLogger)
		if err != nil {
			objLogger.LogEntry(fmt.Sprintf("Failed to extract zip: %s", err), 0, true)
		}
		defer os.RemoveAll(strTempDir)
		objCfg.Attachments = strTempDir
		objLogger.Log(fmt.Sprintf("Using extracted attachments from %s", strTempDir))
	} else {
		if _, err := os.Stat(objCfg.Attachments); os.IsNotExist(err) {
			objLogger.LogEntry(fmt.Sprintf("Attachments path %s does not exist", objCfg.Attachments), 0, true)
		}
	}
	objCfg.Attachments = strings.ReplaceAll(objCfg.Attachments, "\\", "/")

	// Resolve input file
	if objCfg.InFile == "" || *bPrompt {
		objCfg.InFile = getInput("Please enter the path to the file to be processed: ")
	}
	if objCfg.InFile == "" {
		objLogger.LogEntry("No input file provided, exiting", 0, true)
	}
	if !strings.HasSuffix(strings.ToLower(objCfg.InFile), ".csv") {
		objLogger.LogEntry(fmt.Sprintf("Only CSV files supported, got: %s", objCfg.InFile), 0, true)
	}

	// Handle kennitala for mileage entries
	strKennitala := ""
	if strEmpLower != "name" {
		strKennitala = getInput("Please enter the kennitala of the user: ")
		for !ValidateKT(strKennitala) {
			strKennitala = getInput("Invalid kennitala. Please enter a valid kennitala: ")
		}
	}

	// Initialize API client and CSV handler
	objAPI := NewAPIClient(objCfg.Proxy, iTimeOut, iMinQuiet, objLogger)
	objCSV := NewCSVHandler(objCfg.CSVDelim, objLogger)

	// Build headers
	dictHeader := map[string]string{
		"Api-Version": "alpha",
		"Application": strScriptName,
		"User-Agent":  fmt.Sprintf("Go/%s", strScriptName),
	}

	// Authenticate
	dictAuthBody := map[string]string{
		"clientId":     objCfg.ClientID,
		"clientSecret": objCfg.ClientSecret,
	}
	objLogger.Log("Requesting access token")
	objResp := objAPI.MakeAPICall(objCfg.BaseURL+"auth/token", dictHeader, "post", dictAuthBody, nil, "", "")
	if !objResp.bSuccess {
		objLogger.LogEntry(fmt.Sprintf("Failed to get access token: %s", objResp.strError), 0, true)
	}

	// Extract access token
	dictAuthResp, ok := objResp.objData.(map[string]any)
	if !ok {
		objLogger.LogEntry("Unexpected auth response format", 0, true)
	}
	strAccessToken, ok := dictAuthResp["accessToken"].(string)
	if !ok {
		objLogger.LogEntry("No accessToken in auth response", 0, true)
	}
	objLogger.Log("Successfully obtained access token")
	dictHeader["Authorization"] = "Bearer " + strAccessToken

	// Fetch accounts
	objResp = objAPI.MakeAPICall(objCfg.BaseURL+"accounting/accounts", dictHeader, "get", nil, nil, "", "")
	if !objResp.bSuccess {
		objLogger.LogEntry(fmt.Sprintf("Failed to fetch accounts: %s", objResp.strError), 0, true)
	}
	lstAccounts, ok := objResp.objData.([]any)
	if !ok {
		objLogger.LogEntry("Unexpected accounts response format", 0, true)
	}
	dictAcctRef := make(map[string]string)
	for _, objAcct := range lstAccounts {
		dictAcct := objAcct.(map[string]any)
		strAcctID := fmt.Sprintf("%v", dictAcct["id"])
		strAcctCode := fmt.Sprintf("%v", dictAcct["code"])
		dictAcctRef[strAcctCode] = strAcctID
	}
	objLogger.Log(fmt.Sprintf("Fetched %d accounts from Payday", len(dictAcctRef)))

	// Read CSV
	lstRows, err := objCSV.ReadCSV(objCfg.InFile)
	if err != nil {
		objLogger.LogEntry(fmt.Sprintf("Failed to read CSV: %s", err), 0, true)
	}

	// Validate account codes
	lstBadAcctCodes := []string{}
	for _, dictRow := range lstRows {
		strCode, exists := dictRow["Category Account Code"]
		if !exists {
			objLogger.LogEntry("Unable to find Category Account Code column", 0, true)
		}
		if _, found := dictAcctRef[strCode]; !found {
			lstBadAcctCodes = append(lstBadAcctCodes, strCode)
		}
	}
	if len(lstBadAcctCodes) > 0 {
		objLogger.LogEntry(fmt.Sprintf("Unknown account codes: %v", lstBadAcctCodes), 0, true)
	}

	// Fetch payment types
	objResp = objAPI.MakeAPICall(objCfg.BaseURL+"expenses/paymenttypes", dictHeader, "get", nil, nil, "", "")
	if !objResp.bSuccess {
		objLogger.LogEntry(fmt.Sprintf("Failed to fetch payment types: %s", objResp.strError), 0, true)
	}
	lstPayTypes, ok := objResp.objData.([]any)
	if !ok {
		objLogger.LogEntry("Unexpected payment types response format", 0, true)
	}

	// Payment type selection
	fmt.Println("Please select a payment type from the list below")
	fmt.Println("ID: Name (Description)")
	for iIndex, objPT := range lstPayTypes {
		dictPT := objPT.(map[string]any)
		fmt.Printf("%d: %v (%v)\n", iIndex, dictPT["title"], dictPT["description"])
	}
	strPayType := getInput("Please enter the payment type ID: ")
	if !isInt(strPayType) {
		objLogger.LogEntry("Payment type ID must be an integer", 0, true)
	}
	iPayType := 0
	fmt.Sscanf(strPayType, "%d", &iPayType)
	if iPayType < 0 || iPayType >= len(lstPayTypes) {
		objLogger.LogEntry(fmt.Sprintf("Payment type ID must be between 0 and %d", len(lstPayTypes)-1), 0, true)
	}
	strPayTypeID := fmt.Sprintf("%v", lstPayTypes[iPayType].(map[string]any)["id"])
	objLogger.Log(fmt.Sprintf("Payment type ID %d: %s was selected", iPayType, strPayTypeID))

	// Process expense entries
	strURL := objCfg.BaseURL + "expenses"
	strEntryID := ""
	var dictBody map[string]any
	var lstFiles map[string]string
	var lstBadEntryIDs []string

	submitEntry := func() {
		objResp := objAPI.MakeAPICall(strURL, dictHeader, "post", dictBody, lstFiles, "", "")
		objLogger.LogEntry(fmt.Sprintf("APIResp success: %v", objResp.bSuccess), 5, false)
		if !objResp.bSuccess {
			objLogger.Log(fmt.Sprintf("Failed entry %s: %s", strEntryID, objResp.strError))
			lstBadEntryIDs = append(lstBadEntryIDs, strEntryID)
		} else {
			objLogger.Log(fmt.Sprintf("Successfully submitted entry %s", strEntryID))
		}
	}

	for _, dictRow := range lstRows {
		objLogger.LogEntry(fmt.Sprintf("Processing entry#: %s is reimbursable: %s",
			dictRow["Entry Number"], dictRow["Is Reimbursable"]), 5, false)

		if dictRow["Is Reimbursable"] == "false" {
			continue
		}

		strAcctID := dictAcctRef[dictRow["Category Account Code"]]
		strDescription := dictRow["Expense Description"]
		strTaxPct := dictRow["Tax Percentage"]
		if strTaxPct == "" {
			strTaxPct = "0.0"
		}

		dictLine := map[string]any{
			"quantity":              1,
			"description":           strDescription,
			"unitPriceIncludingVat": parseFloat(dictRow["Expense Total Amount (in Reimbursement Currency)"]),
			"vatPercentage":         parseFloat(strTaxPct),
			"accountId":             strAcctID,
		}

		if strEntryID == dictRow["Entry Number"] {
			// Additional line on same entry
			dictBody["lines"] = append(dictBody["lines"].([]any), dictLine)
		} else {
			// Submit previous entry if there was one
			if strEntryID != "" {
				submitEntry()
			}

			strEntryID = dictRow["Entry Number"]
			objLogger.Log(fmt.Sprintf("Working on: %s - Entry %s - Vendor: %s",
				dictRow["Expense Description"], strEntryID, dictRow["Merchant Name"]))

			// Gather attachments
			lstFiles = make(map[string]string)
			lstAttachments := ListAttachments(objCfg.Attachments, strEntryID+"*")
			for iIndex, strFile := range lstAttachments {
				strFilePath := objCfg.Attachments + "/" + strFile
				if _, err := os.Stat(strFilePath); err == nil {
					lstFiles[fmt.Sprintf("attachment%d", iIndex)] = strFilePath
				} else {
					objLogger.LogEntry(fmt.Sprintf("Unable to find attachment file %s", strFilePath), 2, true)
				}
			}

			// Build body
			dictBody = map[string]any{
				"status":      "PAID",
				"creditor":    map[string]any{},
				"date":        dictRow["Expense Item Date"],
				"deductible":  objCfg.Deductible,
				"paidDate":    dictRow["Expense Item Date"],
				"paymentType": map[string]any{"id": strPayTypeID},
				"reference":   dictRow["Report Number"],
				"lines":       []any{dictLine},
			}

			dictCreditor := dictBody["creditor"].(map[string]any)
			if dictRow["Mileage Type"] == "NonMileage" {
				dictCreditor["Name"] = dictRow["Merchant Name"]
				dictCreditor["ssn"] = dictRow["Expense.CF.Kennitala"]
			} else {
				if strings.ToLower(objCfg.EmployeeID) == "name" {
					dictCreditor["Name"] = dictRow["Employee Name"]
					dictCreditor["ssn"] = dictRow["Employee Number"]
				} else {
					dictCreditor["ssn"] = strKennitala
				}
				fDistance := parseFloat(dictRow["Distance"])
				fMileage := parseFloat(dictRow["Mileage Rate"])
				strDescription = fmt.Sprintf("Mileage for %s - %.2f %s @ %.0f kr/km",
					dictRow["Vehicle Name"], fDistance, dictRow["Mileage Unit"], fMileage)
				dictBody["comment"] = dictRow["Expense Description"]
			}
			dictBody["creditor"] = dictCreditor
		}
	}

	// Submit last entry
	if strEntryID != "" {
		submitEntry()
	}

	// Write failed entries if any
	if len(lstBadEntryIDs) == 0 {
		objLogger.Log("All entries processed successfully")
	} else {
		objLogger.Log(fmt.Sprintf("Issues with the following entry IDs: %v", lstBadEntryIDs))
		if err := objCSV.WriteFailedEntries(objCfg.InFile, lstRows, lstBadEntryIDs); err != nil {
			objLogger.Log(fmt.Sprintf("Failed to write failed entries file: %s", err))
		}
	}

	objLogger.Log(fmt.Sprintf("Done processing file %s", objCfg.InFile))
}

func chkdir(strDir string) bool {
	if _, err := os.Stat(strDir); os.IsNotExist(err) {
		if err := os.MkdirAll(strDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Unable to create directory %s: %s\n", strDir, err)
			return false
		}
	}
	return true
}

func getInput(strPrompt string) string {
	fmt.Print(strPrompt)
	objScanner := bufio.NewScanner(os.Stdin)
	objScanner.Scan()
	return strings.TrimSpace(objScanner.Text())
}

func isInt(strVal string) bool {
	if strVal == "" {
		return false
	}
	for _, c := range strVal {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func parseFloat(strVal string) float64 {
	fVal, err := strconv.ParseFloat(strings.TrimSpace(strVal), 64)
	if err != nil {
		return 0.0
	}
	return fVal
}

func ListAttachments(strDirectory string, strPattern string) []string {
	var lstFiles []string
	objEntries, err := os.ReadDir(strDirectory)
	if err != nil {
		return lstFiles
	}
	for _, objEntry := range objEntries {
		if !objEntry.IsDir() {
			if matchPattern(objEntry.Name(), strPattern) {
				lstFiles = append(lstFiles, objEntry.Name())
			}
		}
	}
	return lstFiles
}

func matchPattern(strName string, strPattern string) bool {
	if !strings.Contains(strPattern, "*") {
		return strName == strPattern
	}
	strPrefix := strPattern[:strings.Index(strPattern, "*")]
	return strings.HasPrefix(strName, strPrefix)
}
