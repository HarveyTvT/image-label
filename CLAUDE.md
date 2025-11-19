# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A web-based image labeling tool built with Go. The application provides a clean interface for categorizing images into predefined labels, with automatic organization into labeled subfolders and result export.

**Key Files:**
- `main.go` - HTTP server, handlers, and core logic
- `templates/index.html` - Web UI with AJAX-based labeling
- `labels.txt` - Label definitions (one per line)
- `images/` - Working directory for unlabeled images
- `result.zip` - Exported results (created on exit)

## Common Commands

### Building and Running
```bash
go run main.go          # Run the application (http://localhost:18081)
go build                # Build the binary
make compile            # Cross-compile for multiple platforms (Linux, Windows, macOS)
```

### Development
```bash
go fmt ./...            # Format all Go files
go vet ./...            # Run Go vet for static analysis
```

### Usage Workflow
1. Place unlabeled images in `images/` directory
2. Start the server: `go run main.go`
3. Open browser to http://localhost:18081
4. Click label buttons (1-6) to categorize images
5. Press Ctrl+C to stop server and create `result.zip`

## Architecture

### Application Flow
1. **Startup**: Creates `images/` directory, loads labels from `labels.txt`, sets up signal handlers
2. **Runtime**: Serves paginated image grid (10 per page), handles AJAX labeling requests
3. **Shutdown**: Captures interrupt signal (Ctrl+C), creates `result.zip` with all labeled subfolders

### Data Model
- `PageData`: Template data with images, pagination, and label choices
- `LabelChoice`: Label index (1-6) and text from `labels.txt`

### Key Functions
- `loadLabelChoices()`: Reads and parses `labels.txt` at startup
- `homeHandler()`: Renders main page with paginated images
- `labelHandler()`: Processes labeling requests, moves images to subfolders
- `createResultZip()`: Packages all label subfolders into `result.zip`
- `moveFile()`: Moves images (uses rename if same filesystem, copy+delete otherwise)

### Label System
Labels are defined in `labels.txt` (one per line). Each label gets:
- Index: 1-based position in file
- Text: Display name (supports Unicode/Chinese)
- Subfolder: `images/{label_text}/` created on first use

Example `labels.txt`:
```
正常
名人
色情
暴力
政治敏感
其他
```

### Frontend (templates/index.html)
- Responsive grid layout with image cards
- AJAX submission for instant feedback
- Removes labeled images from UI without page reload
- Auto-reloads when page is empty

### Port Configuration
Server runs on port 18081 (configurable in `main.go` line 78)
