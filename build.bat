set GOOS=darwin&& set GOARCH=arm64&& go build -o ZohoExpense2Payday_mac-arm .
set GOOS=darwin&& set GOARCH=amd64&& go build -o ZohoExpense2Payday_mac-intel .
set GOOS=linux&& set GOARCH=arm64&& go build -o ZohoExpense2Payday_linux-arm .
set GOOS=linux&& set GOARCH=amd64&& go build -o ZohoExpense2Payday_linux-amd64 .
set GOOS=windows&& set GOARCH=amd64&& go build -o ZohoExpense2Payday-amd64.exe .
set GOOS=windows&& set GOARCH=arm64&& go build -o ZohoExpense2Payday-arm64.exe .