# Azubiheft.de MCP Server

MCP (Model Context Protocol) Server for Azubiheft.de integration with Claude Desktop and Cursor.

## 📋 Prerequisites

- macOS
- Go 1.21 or higher
- Claude Desktop App or Cursor IDE

## 🚀 Installation (macOS)

### 1. Compile Server

```bash
make build
```

This creates the binary at `bin/azubiheft-mcp-server`.

### 2. Configure Claude Desktop / Cursor

Edit the config file:

**Claude Desktop:**
```bash
open ~/Library/Application\ Support/Claude/claude_desktop_config.json
```

**Cursor:**
```bash
open ~/.cursor/mcp.json
```

Add the following configuration:

```json
{
  "mcpServers": {
    "azubiheft": {
      "command": "/Users/konrad.maedler/GolandProjects/Azubiheft.deMCP/azubiheft-api/bin/azubiheft-mcp-server",
      "args": [],
      "env": {
        "AZUBIHEFT_USERNAME": "your_username@email.de",
        "AZUBIHEFT_PASSWORD": "your_password"
      }
    }
  }
}
```

**Important:** Adjust the `command` path to match your actual installation path!

### 3. Restart Application

Quit the application completely (Cmd+Q) and restart it.

## 📚 Usage

After restart, you can use commands like:

- `"Show me my subjects at Azubiheft"`
- `"Create a report for today: Subject Company, 8 hours, Web development"`
- `"Show me the report from 2025-01-15"`
- `"Delete all reports from 2025-11-04"`

## 🔧 Development

### Project Structure

```
.
├── cmd/server/          # Main entry point
├── internal/
│   ├── azubiheft/       # Azubiheft.de API Client
│   ├── mcp/            # MCP Server implementation
│   └── server/         # Service layer (tool implementations)
├── bin/                # Compiled binary
└── Makefile           # Build commands
```

### Build Commands

```bash
make build    # Compile the server
make clean    # Delete the binary
```

## 📄 License

See [LICENSE](LICENSE) file.
