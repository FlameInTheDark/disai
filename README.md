# DisAI - Discord AI Bot

DisAI is a Discord bot that integrates with Ollama to provide AI capabilities to your Discord server. It allows users to interact with AI models through a simple chat interface.

## Features

- Discord integration with slash commands
- AI chat capabilities using Ollama models
- Support for function calling through Model Control Plane (MCP)
- Customizable system and user templates
- Load balancing across multiple Ollama servers
- User whitelisting for access control (TODO)

## Prerequisites

- Go 1.18 or higher
- Discord bot token
- Ollama server(s) running with your desired models
- MCP server (optional, for function calling)

## Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/FlameInTheDark/disai.git
   cd disai
   ```

2. Build the application:
   ```bash
   go build -o disai ./cmd/disai
   ```

3. Create a configuration file (see Configuration section below)

4. Run the application:
   ```bash
   ./disai --config ./config.yaml
   ```

## Configuration

Create a `config.yaml` file based on the provided `config.example.yaml`:

```yaml
token: "your_discord_application_token"
model: "model_name"  # e.g., "qwen3:8b"
whitelist:
  - user_id_1  # Discord user IDs allowed to use the bot
  - user_id_2
ollamaServers:
  - "http://localhost:11434"  # List of Ollama server URLs
  - "http://other-server:11434"
mcpServers:
  general: "http://localhost:8089"  # MCP server URL
templates:
  system: "./system.tmpl"  # Path to system template
  user: "./user.tmpl"  # Path to user template
```

### Templates

The bot uses two template files to format messages sent to the AI model:

1. **System Template** (`system.tmpl`): Defines the AI's persona and behavior
2. **User Template** (`user.tmpl`): Formats user messages and provides instructions

You can customize these templates to change the AI's behavior and response format.

## Usage

Once the bot is running and added to your Discord server, you can interact with it using the `/chat` command:

```
/chat message: What's the weather like today?
```

The bot will process your message through the AI model and respond with the AI's reply.

## Discord Bot Setup

1. Create a new application at the [Discord Developer Portal](https://discord.com/developers/applications)
2. Create a bot for your application
3. Enable the "Message Content Intent" under the Bot settings
4. Generate a bot token and add it to your configuration file
5. Use the OAuth2 URL Generator to create an invite link with the following permissions:
   - Scopes: `bot`, `applications.commands`
   - Bot Permissions: `Send Messages`, `Use Slash Commands`
6. Invite the bot to your server using the generated link

## Development

### Project Structure

- `cmd/disai`: Main application code
- `cmd/tool`: Additional tools (MCP)
- `internal/config`: Configuration handling
- `internal/mcp`: Model Control Plane client
- `internal/model`: AI model integration

### Building from Source

```bash
go build -o disai ./cmd/disai
```

### TODO:
- [ ] Add message queue for Ollama server load balancing
- [ ] Add user whitelisting