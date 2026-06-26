# Zoho Expense 2 Payday

This is an application that takes an expense report exported as a csv out of Zoho Expense and imports it into the Payday accounting system, turning every line item in the expense report and turns it into an expense item in Payday, with the invoice/receipt attached. The initial version only offers a command-line interface, so it must be run from a terminal. A graphical user interface is planned.

You need to download both an attachment zip file and the csv export of the report. You can unzip the content and provide the folder path, or just provide the zip file directly.

## Preparations

### Downloading

To download this, go to https://github.com/siggib007/ZohoExpense2Payday/releases (if you are reading this on GitHub, it's the Release heading on the right-hand side, under About project), find the latest release, and expand the Assets header. In there, find the binary that suits your platform.

#### Windows

For 90% of Windows users, you'll need -am64.exe. It should work on ARM platforms as well. If you want to be sure, check settings -> system -> about, the system type line. will show either an x64-based processor or an ARM-based processor. If you are on an ARM-based processor, -arm64.exe will work better. 

#### Linux

Again, most Linux machines run x64 processors; notable exceptions include single-board computers like the Raspberry Pi. Run `uname -m` in the terminal. If the output is x86_64, you need the linux-amd64 executable; it does not matter what flavor of Linux you have, all that matters is what sort of system is running your Linux flavor. If it is not x86_64 you'll need the arm package. BTW x86_32 is not supported

#### macOS

Mac Download:
- Apple Silicon (M1/M2/M3/M4) → ZohoExpense2Payday_mac_arm
  (Most Macs bought after 2020)
- Intel → ZohoExpense2Payday_mac-intel
  (Macs bought before 2021)

Not sure which you have?
Apple menu → About This Mac → look for "Apple M1/M2/M3" or "Intel"

### Zoho Expense Preparations

Start by ensuring the accounting codes for each of your active categories match the account codes in the Payday chart of accounts. If those don't line up or an account code is missing, the application will abort.

To provide an Icelandic Tax ID (aka SSN) for Payday expense items, you need to create a custom expense field called Kennitala. The field does not need to be populated, but the application expects it to exist.

Make sure you have an export template in Zoho Expense that provides the following fields. It might be a good idea to create a new template called Payday with exactly these fields.

- Entry Number
- Is Reimbursable
- Category Account Code
- Expense Description
- Tax Percentage
- Expense Total Amount (in Reimbursement Currency)
- Merchant Name
- Expense Item Date
- Report Number
- Mileage Type
- Expense.CF.Kennitala

If you are using mileage reporting, you need the following as well

- Employee Name
- Employee Number
- Distance
- Mileage Rate
- Vehicle Name
- Mileage Unit

### Payday preparations

The only thing you need to do specifically in Payday, in addition to lining up account codes with Zoho Expense Account codes as mentioned above, is to obtain API credentials and the API URL. If you have both a developer instance and a production instance, you would need to do this for both. You do this in your company setting, on the integration tab (type Payday API). There is also a link there with further documentation.

### Application preparation

Before you can actually run the application, you have to set up the configuration file. Using a simple text editor like Notepad, create a new file named ZohoExpense2payday.ini with the following lines. Anything after # is just an explanation; leave those out of the actual file. The only mandatory lines are the API credentials (ID and secret). Everything else can be provided at run time.

```ini
Environment = production # a flag to indicate if this is intended for test, dev or production
API_URL=https://api.payday.is/ # The URL provided by Payday documentation
CLIENT_ID=redacted # The Client ID you got when creating the API credentials
CLIENT_SECRET=topsecret # The Client secret you got when creating the API credentials
ATTACHMENTS=C:/er/Attach2.zip # If you want to automatically use a specific folder or zip file, provide it here, otherwise provide it on the command line.
IN_FILE=C:/test/Expense_Report.csv # if you want to automatically use a specific csv file, provide it here, otherwise provide it on the command line.
EMPLOYEE_ID=name # name | kt - Employee name in Zoho Expense Report, or prompt for kennitala
PROXY=127.0.0.1:8080 # Proxy for API calls, if needed
```

## Execution

Here is a summary of what is possible at run time

``` cmd
ZohoExpense2Payday.exe -help
Usage of ZohoExpense2Payday.exe:
  -a string
        Path to attachments directory or zip file
  -c string
        Path to configuration file (default "ZohoExpense2Payday.ini" in your current directory)
  -d string
        Is VAT deductible? True/False. Default True
  -e string
        Employee identification: name, kt or kennitala. Default True
  -i string
        Path to expense CSV file to be processed
  -l string
        Path to log file (default "D:/temp/Logs/ZohoExpense2Payday-2026-06-26-12-40-04.log")
  -p    Prompt for input file
  -u string
        Base URL for API calls
  -v int
        Verbosity level (1-5) (default 1)
  -x string
        Proxy for API calls
```

A typical execution will be something like

``` cmd
ZohoExpense2Payday.exe -a c:\download\ER-00026.zip -i c:\download\Expense_Report.csv
```

The application will prompt you for a couple of things and print the status as it processes each line.
