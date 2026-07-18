package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/siggib007/goutils/apiclient"
	"github.com/siggib007/goutils/kennitala"
	"github.com/siggib007/goutils/logger"
	"github.com/siggib007/goutils/utils"
)

func main() {
	// Create default base paths
	objPaths, err := utils.BasePaths()
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannot determine base paths: "+err.Error())
		os.Exit(3)
	}
	strScriptName := objPaths.AppName

	// Load config — three tier: INI -> env vars -> CLI flags
	objCfg := defaultConfig()

	// CLI flags
	strInputFile := flag.String("i", "", "Path to expense CSV file to be processed")
	bPrompt := flag.Bool("p", false, "Prompt for input file")
	strAttachments := flag.String("a", "", "Path to attachments directory or attachment zip file")
	bDeductible := flag.Bool("d", objCfg.Deductible, "Is VAT deductible? True/False. Default: True")
	iVerbose := flag.Int("v", 1, "Verbosity level (1-5)")
	strConfFile := flag.String("c", objPaths.DefConf, "Path to configuration file, defaults to file with same name as the application in the application directory.")
	strBaseURL := flag.String("u", "", "Base URL for API calls")
	strEmployee := flag.String("id", "name", "Employee identification for milage expenses: name, kt or kennitala. Default: name")
	bUseEnv := flag.Bool("e", false, "Indicates not to try to load config file, only use environment variables")
	strProxy := flag.String("x", "", "Proxy for API calls")
	iTimeout := flag.Int("t", objCfg.TimeOut, "Timeout value on API calls, number of seconds")
	flag.Parse()

	fmt.Print("This is a script to transfer expense items from Zoho Expense to Payday.\n")
	fmt.Printf("Running from: %s\n", objPaths.ExeDir)
	fmt.Printf("The time now is %s\n", time.Now().Format("Monday 02 January 2006 15:04:05"))
	fmt.Printf("Logs saved to %s\n", objPaths.DefLogFile)

	// Initialize logger
	objLogger, err := logger.NewLogger(objPaths.DefLogFile, *iVerbose)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create log file: %s\n", err)
		os.Exit(1)
	}

	defer objLogger.Close()
	defer objLogger.RecoverAbort()
	strScriptHost, err := os.Hostname()
	if err != nil {
		objLogger.Log("Failed to determine hostname: " + err.Error())
		strScriptHost = "HOSTNAME-LOOKUP-FAILED"
	}

	objLogger.Log(fmt.Sprintf("Starting up script %s on %s", strScriptName, strScriptHost))
	objLogger.Log(fmt.Sprintf("Verbosity set to %d", *iVerbose))

	utils.ValidateConfPath(objLogger, strConfFile, *bUseEnv, *objPaths)

	objCfg.Verbose = *iVerbose

	if *strConfFile != "env" {
		if err := parseINI(*strConfFile, &objCfg); err != nil {
			objLogger.Log(fmt.Sprintf("Could not read config file %s: %s", *strConfFile, err))
		}
	} else {
		objLogger.Log("Not loading configuration file, relying on environment variables. Make sure they are set correctly")
	}
	applyEnvVars(&objCfg)

	dictFlagsSet := make(map[string]bool)
	flag.Visit(func(objFlag *flag.Flag) {
		dictFlagsSet[objFlag.Name] = true
	})

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

	if dictFlagsSet["d"] {
		objCfg.Deductible = *bDeductible
	}
	if dictFlagsSet["t"] {
		objCfg.TimeOut = *iTimeout
	}

	// Validate required config
	if objCfg.BaseURL == "" || objCfg.ClientID == "" || objCfg.ClientSecret == "" {
		objLogger.LogEntry("No URL or API auth config, exiting", 0, true)
	}

	// Validate employee ID type
	strEmpLower := strings.ToLower(objCfg.EmployeeID)
	if strEmpLower != "name" && strEmpLower != "kt" && strEmpLower != "kennitala" {
		objLogger.LogEntry("Employee ID must be either name, kt or kennitala", 0, true)
	}

	if _, err := os.Stat(objCfg.Attachments); os.IsNotExist(err) {
		objLogger.LogEntry(fmt.Sprintf("Attachments path %s does not exist", objCfg.Attachments), 0, false)
		strInput := utils.GetInput("Please provide a new path for attachments: ")
		if _, err := os.Stat(strInput); os.IsNotExist(err) {
			objLogger.LogEntry(fmt.Sprintf("Attachments path %s does not exist", objCfg.Attachments), 0, true)
		}
		objCfg.Attachments = strInput
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
	}

	// Resolve input file
	if objCfg.InFile == "" || *bPrompt {
		objCfg.InFile = utils.GetInput("Please enter the path to the file to be processed: ")
	}
	if objCfg.InFile == "" {
		objLogger.LogEntry("No input file provided, exiting", 0, true)
	}
	if !strings.EqualFold(filepath.Ext(objCfg.InFile), ".csv") {
		objLogger.LogEntry(fmt.Sprintf("Only CSV files supported, got: %s", objCfg.InFile), 0, true)
	}

	// Handle kennitala for mileage entries
	strKennitala := ""
	if strEmpLower != "name" {
		strKennitala = utils.GetInput("Please enter the kennitala of the user: ")
		for !kennitala.ValidateKT(strKennitala) {
			strKennitala = utils.GetInput("Invalid kennitala. Please enter a valid kennitala: ")
		}
	}

	// Initialize API client and CSV handler
	objAPI := apiclient.NewAPIClient(objCfg.Proxy, objCfg.TimeOut, objCfg.MinQuiet, objLogger)
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

	// Build URL string
	dictMyParams := make(map[string]string)
	strURL := apiclient.BuildURL(objCfg.BaseURL, "auth/token", dictMyParams)

	objCallOptions := apiclient.APICallOptions{}
	objCallOptions.URL = strURL
	objCallOptions.Header = dictHeader
	objCallOptions.Method = "POST"
	objCallOptions.Payload = dictAuthBody

	objLogger.Log("Requesting access token")
	objResp := objAPI.MakeAPICall(objCallOptions)
	if !objResp.Success {
		objLogger.LogEntry(fmt.Sprintf("Failed to get access token: %s", objResp.Error), 0, true)
	}

	// Extract access token
	dictAuthResp, ok := objResp.Data.(map[string]any)
	if !ok {
		objLogger.LogEntry("Unexpected auth response format", 0, true)
	}
	strAccessToken, ok := dictAuthResp["accessToken"].(string)
	if !ok {
		objLogger.LogEntry("No accessToken in auth response", 0, true)
	}
	objLogger.Log("Successfully obtained access token")
	dictHeader["Authorization"] = "Bearer " + strAccessToken
	objCallOptions.Header = dictHeader

	// Build URL string
	dictMyParams = make(map[string]string)
	strURL = apiclient.BuildURL(objCfg.BaseURL, "accounting/accounts", dictMyParams)
	objCallOptions.URL = strURL
	objCallOptions.Method = "GET"

	// Fetch accounts
	objResp = objAPI.MakeAPICall(objCallOptions)
	if !objResp.Success {
		objLogger.LogEntry(fmt.Sprintf("Failed to fetch accounts: %s", objResp.Error), 0, true)
	}
	lstAccounts, ok := objResp.Data.([]any)
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
	// Build URL string
	dictMyParams = make(map[string]string)
	strURL = apiclient.BuildURL(objCfg.BaseURL, "expenses/paymenttypes", dictMyParams)
	objCallOptions.URL = strURL

	// Fetch payment types
	objResp = objAPI.MakeAPICall(objCallOptions)
	if !objResp.Success {
		objLogger.LogEntry(fmt.Sprintf("Failed to fetch payment types: %s", objResp.Error), 0, true)
	}
	lstPayTypes, ok := objResp.Data.([]any)
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
	strPayType := utils.GetInput("Please enter the payment type ID: ")
	iPayType, err := strconv.Atoi(strPayType)
	if err != nil {
		objLogger.LogEntry("Payment type ID must be an integer", 0, true)
	}
	if iPayType < 0 || iPayType >= len(lstPayTypes) {
		objLogger.LogEntry(fmt.Sprintf("Payment type ID must be between 0 and %d", len(lstPayTypes)-1), 0, true)
	}
	strPayTypeID := fmt.Sprintf("%v", lstPayTypes[iPayType].(map[string]any)["id"])
	objLogger.Log(fmt.Sprintf("Payment type ID %d: %s was selected", iPayType, strPayTypeID))

	if objCfg.Environment != "" {
		objLogger.Log(fmt.Sprintf("Ready to start processing %v", objCfg.Environment))
		strConfirmation := utils.GetInput("Please enter the environment name to confirm ready to proceed: ")
		if strConfirmation != objCfg.Environment {
			objLogger.LogEntry("Confirmation doesn't match, unable to proceed", 0, true)
		}
	}

	// Build URL string
	dictMyParams = make(map[string]string)
	strURL = apiclient.BuildURL(objCfg.BaseURL, "expenses", dictMyParams)
	objCallOptions.URL = strURL

	// Process expense entries
	strEntryID := ""
	var dictBody map[string]any
	var lstFiles map[string]string
	var lstBadEntryIDs []string

	submitEntry := func() {
		objResp := objAPI.MakeAPICall(objCallOptions)
		objLogger.LogEntry(fmt.Sprintf("APIResp success: %v", objResp.Success), 5, false)
		if !objResp.Success {
			objLogger.Log(fmt.Sprintf("Failed entry %s: %s", strEntryID, objResp.Error))
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
			"unitPriceIncludingVat": utils.ParseFloat(dictRow["Expense Total Amount (in Reimbursement Currency)"]),
			"vatPercentage":         utils.ParseFloat(strTaxPct),
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
			objLogger.Log(fmt.Sprintf("Working on: %s - Entry %s - Milage type: %v - Vendor: %s",
				dictRow["Expense Description"], strEntryID, dictRow["Mileage Type"], dictRow["Merchant Name"]))

			// Gather attachments
			lstFiles = make(map[string]string)
			lstAttachments := ListAttachments(objCfg.Attachments, strEntryID+"*")
			for iIndex, strFile := range lstAttachments {
				strFilePath := filepath.Join(objCfg.Attachments, strFile)
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
				fDistance := utils.ParseFloat(dictRow["Distance"])
				fMileage := utils.ParseFloat(dictRow["Mileage Rate"])
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

func ListAttachments(strDirectory string, strPattern string) []string {
	var lstFiles []string
	objEntries, err := os.ReadDir(strDirectory)
	if err != nil {
		return lstFiles
	}
	for _, objEntry := range objEntries {
		if !objEntry.IsDir() {
			if utils.MatchPattern(objEntry.Name(), strPattern) {
				lstFiles = append(lstFiles, objEntry.Name())
			}
		}
	}
	return lstFiles
}
