package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

type UI struct {
	objApp       fyne.App
	objWindow    fyne.Window
	objLogger    *Logger
	objCfg       *Config
	strStatefile string

	// Run tab widgets
	objInputFile   *widget.Entry
	objAttachments *widget.Entry
	objKennitala   *widget.Entry
	objEmployeeID  *widget.Select
	objDeductible  *widget.Check
	objLogOutput   *widget.Entry
	objRunBtn      *widget.Button

	// Settings tab widgets
	objBaseURL      *widget.Entry
	objClientID     *widget.Entry
	objClientSecret *widget.Entry
	objProxy        *widget.Entry
	objCSVDelim     *widget.Entry
	objIniSelect    *widget.Select
	lstIniFiles     []string
}

func NewUI(objCfg *Config, objLogger *Logger, strBaseDir string) *UI {
	return &UI{
		objApp:       app.New(),
		objCfg:       objCfg,
		objLogger:    objLogger,
		strStatefile: filepath.Join(strBaseDir, "ZohoExpense2Payday.state"),
	}
}

func (u *UI) Run() {
	u.objWindow = u.objApp.NewWindow("ZohoExpense2Payday")
	u.objWindow.Resize(fyne.NewSize(700, 600))

	// Build tabs
	objTabs := container.NewAppTabs(
		container.NewTabItem("Run", u.buildRunTab()),
		container.NewTabItem("Settings", u.buildSettingsTab()),
	)

	u.objWindow.SetContent(objTabs)
	u.objWindow.ShowAndRun()
}

func (u *UI) buildRunTab() fyne.CanvasObject {
	// Input file
	u.objInputFile = widget.NewEntry()
	u.objInputFile.SetPlaceHolder("Path to expense CSV file")
	objInputBtn := widget.NewButton("Browse", func() {
		dialog.ShowFileOpen(func(objURI fyne.URIReadCloser, err error) {
			if err != nil || objURI == nil {
				return
			}
			u.objInputFile.SetText(objURI.URI().Path())
			u.saveState()
		}, u.objWindow)
	})

	// Attachments
	u.objAttachments = widget.NewEntry()
	u.objAttachments.SetPlaceHolder("Path to attachments directory or ZIP file")
	objAttachBtn := widget.NewButton("Browse", func() {
		dialog.ShowFolderOpen(func(objURI fyne.ListableURI, err error) {
			if err != nil || objURI == nil {
				return
			}
			u.objAttachments.SetText(objURI.Path())
			u.saveState()
		}, u.objWindow)
	})

	// Employee ID dropdown
	u.objEmployeeID = widget.NewSelect([]string{"name", "kennitala", "kt"}, func(strVal string) {
		u.objKennitala.Hidden = strings.ToLower(strVal) == "name"
		u.objKennitala.Refresh()
	})
	u.objEmployeeID.SetSelected("name")

	// Kennitala field — hidden by default
	u.objKennitala = widget.NewEntry()
	u.objKennitala.SetPlaceHolder("Kennitala")
	u.objKennitala.Hidden = true
	u.objEmployeeID.SetSelected("name")

	// Deductible checkbox
	u.objDeductible = widget.NewCheck("VAT Deductible", nil)
	u.objDeductible.SetChecked(true)

	// Log output area
	u.objLogOutput = widget.NewMultiLineEntry()
	u.objLogOutput.Disable() // readonly
	u.objLogOutput.SetMinRowsVisible(10)

	// Wire logger to UI
	u.objLogger.fnUILog = func(strMsg string) {
		current := u.objLogOutput.Text
		u.objLogOutput.SetText(current + strMsg + "\n")
	}

	// Run button
	u.objRunBtn = widget.NewButton("Run", func() {
		u.objRunBtn.Disable()
		u.objLogOutput.SetText("")
		go u.runProcess()
	})

	// Layout
	objForm := container.NewVBox(
		widget.NewLabel("Input CSV File:"),
		container.NewBorder(nil, nil, nil, objInputBtn, u.objInputFile),
		widget.NewLabel("Attachments:"),
		container.NewBorder(nil, nil, nil, objAttachBtn, u.objAttachments),
		widget.NewLabel("Employee Identification:"),
		u.objEmployeeID,
		u.objKennitala,
		u.objDeductible,
		widget.NewSeparator(),
		u.objRunBtn,
		widget.NewSeparator(),
		widget.NewLabel("Log Output:"),
		u.objLogOutput,
	)

	// Load last state
	u.loadState()

	return objForm
}

func (u *UI) buildSettingsTab() fyne.CanvasObject {
	// INI file selector
	u.lstIniFiles = findIniFiles()
	u.objIniSelect = widget.NewSelect(u.lstIniFiles, func(strVal string) {
		u.loadINIIntoSettings(strVal)
		u.saveState()
	})

	// Settings fields
	u.objBaseURL = widget.NewEntry()
	u.objBaseURL.SetPlaceHolder("https://api.payday.is/")
	u.objClientID = widget.NewEntry()
	u.objClientID.SetPlaceHolder("Client ID")
	u.objClientSecret = widget.NewPasswordEntry()
	u.objClientSecret.SetPlaceHolder("Client Secret")
	u.objProxy = widget.NewEntry()
	u.objProxy.SetPlaceHolder("http://proxy:8080 (optional)")
	u.objCSVDelim = widget.NewEntry()
	u.objCSVDelim.SetText(",")
	u.objCSVDelim.SetPlaceHolder(",")

	// Save button
	objSaveBtn := widget.NewButton("Save Settings", func() {
		u.saveSettings()
	})

	// Layout
	objForm := container.NewVBox(
		widget.NewLabel("Configuration File:"),
		u.objIniSelect,
		widget.NewSeparator(),
		widget.NewLabel("API URL:"),
		u.objBaseURL,
		widget.NewLabel("Client ID:"),
		u.objClientID,
		widget.NewLabel("Client Secret:"),
		u.objClientSecret,
		widget.NewLabel("Proxy (optional):"),
		u.objProxy,
		widget.NewLabel("CSV Delimiter:"),
		u.objCSVDelim,
		widget.NewSeparator(),
		objSaveBtn,
	)

	// Load last used INI
	u.loadLastINI()

	return objForm
}

func (u *UI) loadINIIntoSettings(strPath string) {
	objCfg := defaultConfig()
	if err := parseINI(strPath, &objCfg); err != nil {
		dialog.ShowError(err, u.objWindow)
		return
	}
	u.objBaseURL.SetText(objCfg.BaseURL)
	u.objClientID.SetText(objCfg.ClientID)
	u.objClientSecret.SetText(objCfg.ClientSecret)
	u.objProxy.SetText(objCfg.Proxy)
	u.objCSVDelim.SetText(string(objCfg.CSVDelim))
	*u.objCfg = objCfg
}

func (u *UI) saveSettings() {
	strPath := u.objIniSelect.Selected
	if strPath == "" {
		dialog.ShowInformation("No config selected", "Please select or create a config file first", u.objWindow)
		return
	}

	objFile, err := os.Create(strPath)
	if err != nil {
		dialog.ShowError(err, u.objWindow)
		return
	}
	defer objFile.Close()

	fmt.Fprintf(objFile, "API_URL=%s\n", u.objBaseURL.Text)
	fmt.Fprintf(objFile, "CLIENT_ID=%s\n", u.objClientID.Text)
	fmt.Fprintf(objFile, "CLIENT_SECRET=%s\n", u.objClientSecret.Text)
	fmt.Fprintf(objFile, "PROXY=%s\n", u.objProxy.Text)
	fmt.Fprintf(objFile, "CSV_DELIM=%s\n", u.objCSVDelim.Text)

	dialog.ShowInformation("Saved", fmt.Sprintf("Settings saved to %s", strPath), u.objWindow)
	u.objLogger.Log(fmt.Sprintf("Settings saved to %s", strPath))
}

func findIniFiles() []string {
	var lstFiles []string
	objEntries, err := os.ReadDir(".")
	if err != nil {
		return lstFiles
	}
	for _, objEntry := range objEntries {
		if !objEntry.IsDir() && strings.ToLower(filepath.Ext(objEntry.Name())) == ".ini" {
			lstFiles = append(lstFiles, objEntry.Name())
		}
	}
	return lstFiles
}

func (u *UI) loadLastINI() {
	strState := u.readState("last_ini")
	if strState != "" {
		// verify it still exists
		if _, err := os.Stat(strState); err == nil {
			u.objIniSelect.SetSelected(strState)
			u.loadINIIntoSettings(strState)
			return
		}
	}
	// default to first found
	if len(u.lstIniFiles) > 0 {
		u.objIniSelect.SetSelected(u.lstIniFiles[0])
		u.loadINIIntoSettings(u.lstIniFiles[0])
	}
}

func (u *UI) loadState() {
	if strCSV := u.readState("last_csv"); strCSV != "" {
		u.objInputFile.SetText(strCSV)
	}
	if strAttach := u.readState("last_attachments"); strAttach != "" {
		u.objAttachments.SetText(strAttach)
	}
}

func (u *UI) saveState() {
	dictState := map[string]string{
		"last_csv":         u.objInputFile.Text,
		"last_attachments": u.objAttachments.Text,
		"last_ini":         u.objIniSelect.Selected,
	}
	objFile, err := os.Create(u.strStatefile)
	if err != nil {
		return
	}
	defer objFile.Close()
	for strKey, strVal := range dictState {
		fmt.Fprintf(objFile, "%s=%s\n", strKey, strVal)
	}
}

func (u *UI) readState(strKey string) string {
	objFile, err := os.Open(u.strStatefile)
	if err != nil {
		return ""
	}
	defer objFile.Close()

	objCfg := defaultConfig()
	// reuse ini parser for state file
	objTempCfg := defaultConfig()
	_ = objTempCfg

	// simple key=value read
	objBytes, err := os.ReadFile(u.strStatefile)
	if err != nil {
		return ""
	}
	_ = objCfg
	for _, strLine := range strings.Split(string(objBytes), "\n") {
		strLine = strings.TrimSpace(strLine)
		if strings.HasPrefix(strLine, strKey+"=") {
			return strings.TrimPrefix(strLine, strKey+"=")
		}
	}
	return ""
}

func (u *UI) runProcess() {
	// Re-enable run button when done
	defer func() {
		u.objRunBtn.Enable()
	}()

	// Validate inputs
	if u.objInputFile.Text == "" {
		u.objLogger.Log("Error: No input file selected")
		return
	}
	if u.objAttachments.Text == "" {
		u.objLogger.Log("Error: No attachments path selected")
		return
	}

	// Update config from UI
	u.objCfg.InFile = u.objInputFile.Text
	u.objCfg.Attachments = u.objAttachments.Text
	u.objCfg.EmployeeID = u.objEmployeeID.Selected
	u.objCfg.Deductible = u.objDeductible.Checked

	// Validate kennitala if needed
	if strings.ToLower(u.objCfg.EmployeeID) != "name" {
		strKT := u.objKennitala.Text
		if !ValidateKT(strKT) {
			u.objLogger.Log("Error: Invalid kennitala")
			return
		}
		u.objCfg.InFile = strKT
	}

	u.objLogger.Log("Starting process...")
	u.saveState()

	// Call the core processing function
	runCore(u.objCfg, u.objLogger)
}
