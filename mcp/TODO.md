# MCP Soccer Statistics Server - TODO

## Project Goal
Build an MCP server that can answer questions like "how many goals does Charlotte FC concede per game this season?" by providing tools to search teams, get fixtures, and calculate statistics from API-football.com.

## Information Flow
1. Find team by name → Get team ID
2. Identify current league/season
3. Retrieve match data for team in season
4. Calculate statistics (goals conceded per game)

## Tools Implementation Status

### ✅ Completed
- [x] **search_teams** - Search for teams by name/partial name ✅
  - Implemented with API-football.com integration
  - Supports team name search with optional league/country filters
  - Returns team ID, name, country, venue details
  - Proper error handling and response formatting

### 🚧 In Progress
- [ ] None currently

### 📋 Planned Tools

#### Core Search & Discovery
- [x] **search_teams** - Find teams by name, get team ID and basic info ✅
- [ ] **get_leagues** - List available leagues/competitions, filter by country
- [ ] **get_seasons** - Get available seasons for a league, identify current season

#### Team Data & Statistics  
- [ ] **get_team_fixtures** - Get all matches for a team in a season with results
- [ ] **get_team_statistics** - Get aggregated team stats for a season
- [ ] **get_standings** - League table/standings for context

#### Match Details
- [ ] **get_match_details** - Detailed match information, lineups, events

#### Analysis & Calculations
- [ ] **calculate_team_stats** - Helper to calculate averages from fixture data

## API Research Tasks

### 🔍 Current Research
- [x] **Teams Endpoint Research** - Study API-football.com docs for team search endpoints ✅
  - **Primary endpoint**: `/teams` - Main teams endpoint with search capabilities
  - **Parameters**: `search` (team name), `league`, `season`, `country`
  - **Alternative**: `/teams/{id}` - Get specific team by ID
  - ✅ Response format understood and implemented
  - ✅ Authentication working with X-RapidAPI-Key header

### 📚 Remaining API Research
- [ ] **Map Key Endpoints**:
  - `/teams` - Team search and details
  - `/leagues` - Available competitions  
  - `/fixtures` - Match schedules and results
  - `/standings` - League tables
  - `/teams/statistics` - Team season statistics
- [ ] **Authentication**: API-football.com uses API key in headers
- [ ] **Rate Limiting**: Understand daily/monthly request limits
- [ ] **Response Format**: JSON structure for each endpoint
- [ ] Test endpoint responses and error handling

## Technical Implementation

### Project Setup
- [ ] Set up Bun project structure
- [ ] Install dependencies (MCP SDK, HTTP client)
- [ ] Configure TypeScript
- [ ] Set up API-football.com authentication

### MCP Server Structure
- [ ] Implement MCP server boilerplate
- [ ] Define tool schemas
- [ ] Implement error handling
- [ ] Add logging and debugging

### Testing & Documentation
- [ ] Create test cases for each tool
- [ ] Document tool usage examples
- [ ] Test end-to-end scenarios
- [ ] Performance testing

## Example Questions to Support
- "How many goals does Charlotte FC concede per game this season?"
- "What's Real Madrid's win rate in La Liga this year?"
- "When is Barcelona's next match?"
- "Who are the top scorers in the Premier League?"

## API Research Notes

### Team Search Strategy
- **Primary**: Use `/teams?search={name}` for fuzzy team name matching
- **Fallback**: Search by league if team name search fails
- **Handle variations**: "Charlotte FC", "Charlotte", "CF Charlotte"
- **Response includes**: team ID, name, logo, country, league info

### Expected API Structure
```
GET /teams?search=Charlotte FC
Authorization: X-RapidAPI-Key: {api_key}

Response:
{
  "results": 1,
  "teams": [{
    "team": {
      "id": 1234,
      "name": "Charlotte FC",
      "code": "CHA",
      "country": "USA",
      "founded": 2019,
      "logo": "https://..."
    }
  }]
}
```

## Implementation Notes
- API-football.com provides comprehensive soccer data
- Need to handle team name variations and partial matches
- Consider caching frequently requested data (teams, leagues)
- Plan for different leagues (MLS, Premier League, La Liga, etc.)
- Error handling for team not found scenarios