# Homebrew Tap for Lil

This repository contains the Homebrew formulae for [Lil](https://github.com/pzurek/lil), a lightweight systray app that displays your Linear issues directly in your system tray/menu bar.

## Usage

First, add the tap:

```bash
brew tap pzurek/lil
```

Then install Lil:

```bash
brew install lil
```

## Setup

After installation, you'll need to set your Linear API key:

```bash
export LINEAR_API_KEY=your_linear_api_key
```

For permanent setup, add this line to your shell profile (`~/.bashrc`, `~/.zshrc`, etc.).

## Features

- Displays all active issues assigned to you in Linear
- Groups issues by project with clear separators
- Shows issue details in tooltips (project, due date, assignee, status)
- Opens issues in your browser when clicked
- Automatically refreshes to show the latest issues
- Minimal resource usage 