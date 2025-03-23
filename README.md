# ExpenseTrace

ExpenseTrace is a privacy-focused expense tracking tool that helps you manage your finances without sharing your banking details with third-party services. Built in Go, it provides a simple yet powerful way to track your expenses, categorize them, and generate insightful reports.

## Why ExpenseTrace?

In an era where financial data privacy is increasingly important, ExpenseTrace offers a secure alternative to traditional expense tracking apps. Instead of connecting to your bank accounts or sharing sensitive financial information, ExpenseTrace allows you to:

- Manually import your expenses from CSV or JSON files
- Categorize transactions based on your own rules
- Generate detailed reports and insights
- Access your data through a CLI or web interface
- Keep all your financial data local and private

## Features

- 📊 Interactive TUI (Terminal User Interface) for easy navigation
- 🌐 Web interface for visual data exploration
- 📝 CSV, JSON import functionality
- 📈 Detailed financial reports
- 🔒 Local data storage with SQLite
- 🎨 Beautiful terminal output with color coding

## Data Privacy

ExpenseTrace is designed with privacy in mind:
- All data is stored locally in a SQLite database
- No external API calls or data sharing
- No bank account connections required
- Full control over your financial data

## Installation

### Prerequisites

- Go 1.22 or later
- SQLite

### Building from Source

1. Clone the repository:
```bash
git clone https://github.com/GustavoCaso/expensetrace.git
cd expensetrace
```

2. Build the project:
```bash
go build
```

3. Create a configuration file (`expense.yaml`):
```yaml
db: expenses.db
categories:
  - name: "Groceries"
    pattern: "SUPERMARKET"
  - name: "Transportation"
    pattern: "UBER|TAXI|METRO"
```

### Configuration

The `expense.yaml` file allows you to:
- Set the database location
- Define expense categories and their matching patterns

## Usage

ExpenseTrace provides several commands to help you manage your expenses:

### CLI Commands

#### `tui`
Launches the interactive terminal user interface:
```bash
expensetrace tui
```
The TUI provides a split view showing categories on the left and expenses on the right. Use Tab to switch between views and arrow keys to navigate.

#### `import`
Import expenses from a file:
```bash
expensetrace import -f expenses.csv
```

#### `report`
Generate expense reports:
```bash
expensetrace report [-month MONTH] [-year YEAR] [-v]
```
Options:
- `-month`: Specify month (1-12)
- `-year`: Specify year
- `-v`: Verbose output with detailed expense breakdown

#### `search`
Search for specific expenses:
```bash
expensetrace search -k "keyword" [-v]
```
Options:
- `-k`: Search keyword
- `-v`: Verbose output

#### `category`
Manage expense categories:
```bash
expensetrace category -a inspect|recategorize|migrate [-o output.txt]
```
Options:
- `-a`: Action to perform (inspect, recategorize, or migrate)
- `-o`: Output file for inspection results

#### `web`
Launch the web interface:
```bash
expensetrace web [-p PORT]
```
Options:
- `-p`: Port number (default: 8080)

#### `delete`
Reset the database:
```bash
expensetrace delete
```
⚠️ Warning: This will delete all your expense data!

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.
