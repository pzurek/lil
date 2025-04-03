# Lil

Lil is a lightweight systray app that displays your Linear issues directly in your system tray/menu bar. It allows you to quickly view and access your assigned Linear issues without switching to a browser.

## Features

- Displays all active issues assigned to you in Linear
- Groups issues by project with clear separators
- Shows issue details in tooltips (project, due date, assignee, status)
- Opens issues in your browser when clicked
- Automatically refreshes to show the latest issues
- Minimal resource usage

## Requirements

- Go 1.16 or higher
- Linear API key
- macOS, Linux, or Windows

## Installation

### Using Homebrew (macOS)

```bash
# Install from the Homebrew tap
brew tap pzurek/lil
brew install lil
```

### Using Make

```bash
# Clone the repository
git clone https://github.com/pzurek/lil.git
cd lil

# Build and install
make install
```

### Manual Installation

```bash
# Clone the repository
git clone https://github.com/pzurek/lil.git
cd lil

# Build the application
go build -o lil .

# Move to a directory in your PATH (optional)
mv lil /usr/local/bin/
```

## Setup

1. Get your Linear API key from [Linear Settings → API](https://linear.app/settings/api)
2. Set it as an environment variable:

```bash
export LINEAR_API_KEY=your_linear_api_key
```

For permanent setup, add this line to your shell profile (`~/.bashrc`, `~/.zshrc`, etc.).

## Usage

### Starting the App

```bash
# Run directly
lil

# Or with the API key (if not set in your environment)
LINEAR_API_KEY=your_linear_api_key lil
```

### Using the Menu

- Click on the Lil icon in your system tray/menu bar to see your assigned issues
- Issues are grouped by project with separators between projects
- Hover over an issue to see additional details (project, due date, assignee, status)
- Click on an issue to open it in your default web browser
- Click "Quit" to exit the application

## Development

### Project Structure

```
.
├── assets/                 # Icon and other static assets
├── internal/
│   └── linear/             # Linear API integration
│       └── schema/         # GraphQL schema and generated code
├── main.go                 # Main application code
├── main_test.go            # Tests
└── Makefile                # Build and development scripts
```

### Makefile Commands

- `make build`: Build the application
- `make run`: Build and run the application
- `make test`: Run tests
- `make linear`: Fetch Linear schema and generate code
- `make install`: Install to /usr/local/bin
- `make check`: Run all code quality checks

### GraphQL Schema Generation

This project uses the Linear GraphQL API. To update the schema:

```bash
# Make sure your LINEAR_API_KEY is set
export LINEAR_API_KEY=your_linear_api_key

# Update schema and regenerate code
make linear
```

### Adding Custom Queries

1. Edit `internal/linear/schema/operations.graphql` to add or modify GraphQL queries
2. Run `make linear` to regenerate the client code

## License

[MIT License](LICENSE)

## Acknowledgements

- [Linear](https://linear.app) for their excellent issue tracking system and API
- [systray](https://github.com/getlantern/systray) for the cross-platform systray library
- [genqlient](https://github.com/Khan/genqlient) for GraphQL client generation 