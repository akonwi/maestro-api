# Agents Guide

## Architecture

**maestro-api** is a soccer statistics API with two subprojects:

1. **server/** - HTTP API in Ard language (custom compiled language)
   - Routes: /juice, /bets, /leagues, /matches, /analysis
   - Database: SQLite (maestro.sqlite)
   - Auth: X-Api-Token header check
   - External data: API-football.com via match/predictions/odds modules

2. **mcp/** - Model Context Protocol server in TypeScript + Bun
   - Tools: get-teams, get-fixtures, get-goal-stats, get-predictions, get-odds
   - API client for API-football.com v3

## Build & Test Commands

**MCP subproject:**
- `bun install` - Install dependencies
- `bun run index.ts` - Run MCP server (stdout/stdio transport)

**Server (Ard):**
- Requires `ard` binary (auto-built in Docker)
- `ard run server/main.ard` - Run HTTP server on $PORT (default 8080)
- Docker: `docker build --build-arg GITHUB_TOKEN=<token> -t maestro .`

## Code Style & Conventions

**Ard (server/):**
- Pattern matching for control flow (match/switch)
- Error handling: `try expr -> default_val` or `try expr -> err { error_handler(err) }`
- HTTP responses: use global `res_headers`, return `http::Response{status, body, headers}`
- Module structure: `use maestro/module_name` for imports
- Functions: `fn name(param: Type) ReturnType { ... }`

**TypeScript (mcp/):**
- Strict mode enabled (tsconfig.json)
- Result type pattern: `Result<Data, Error> = OK<Data> | NotOK<Error>`
- Tool definitions: `server.tool(name, description, zod_schema, async_handler)`
- Import syntax: ES modules (type: "module")
- Error handling: fetch errors wrapped in Result type
