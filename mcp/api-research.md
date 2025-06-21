# API-Football.com Research Document

## Overview
Research document for implementing team search functionality in the MCP soccer statistics server. This document outlines the API endpoints, request/response formats, and implementation strategy for the `search_teams` tool.

## API Base Information
- **Base URL**: `https://v3.football.api-sports.io`
- **Authentication**: X-RapidAPI-Key header required
- **Rate Limits**: Varies by subscription plan (typically 100-1000 requests/day for free tier)
- **Response Format**: JSON

## Team Search Endpoints

### Primary Endpoint: `/teams`
**Purpose**: Search for teams by name, league, season, or country

**Request Format**:
```
GET /teams?search={team_name}
Headers:
  X-RapidAPI-Key: {api_key}
  X-RapidAPI-Host: v3.football.api-sports.io
```

**Parameters**:
- `search` (string): Team name or partial name
- `league` (int): League ID to filter results
- `season` (int): Season year (e.g., 2024)
- `country` (string): Country name or code
- `code` (string): Team code (3 letters)
- `venue` (int): Venue/stadium ID

**Expected Response Structure**:
```json
{
  "get": "teams",
  "parameters": {
    "search": "Charlotte FC"
  },
  "errors": [],
  "results": 1,
  "paging": {
    "current": 1,
    "total": 1
  },
  "response": [
    {
      "team": {
        "id": 1234,
        "name": "Charlotte FC",
        "code": "CHA",
        "country": "USA",
        "founded": 2019,
        "national": false,
        "logo": "https://media.api-sports.io/football/teams/1234.png"
      },
      "venue": {
        "id": 5678,
        "name": "Bank of America Stadium",
        "address": "800 South Mint Street",
        "city": "Charlotte",
        "capacity": 75523,
        "surface": "grass",
        "image": "https://media.api-sports.io/football/venues/5678.png"
      }
    }
  ]
}
```

### Alternative Endpoint: `/teams/{id}`
**Purpose**: Get specific team details by ID

**Request Format**:
```
GET /teams/{team_id}
Headers:
  X-RapidAPI-Key: {api_key}
  X-RapidAPI-Host: v3.football.api-sports.io
```

## Implementation Strategy for search_teams Tool

### 1. Input Processing
- Accept team name as string parameter
- Handle variations: "Charlotte FC", "Charlotte", "CF Charlotte"
- Normalize input (trim, case-insensitive matching)

### 2. Search Logic
```
1. Primary search: Use exact team name with /teams?search={name}
2. If no results: Try partial matching with common variations
3. If still no results: Return empty result with suggestion to try different name
4. Return top matches (limit to 5-10 results to avoid overwhelming)
```

### 3. Response Processing
- Extract relevant team information (id, name, country, league)
- Format for MCP tool response
- Include venue information if available
- Handle API errors gracefully

### 4. Error Handling
- **404/No Results**: Team not found, suggest similar teams
- **401**: Invalid API key
- **403**: Rate limit exceeded
- **500**: API server error
- **Network errors**: Timeout, connection issues

## MCP Tool Schema for search_teams

```json
{
  "name": "search_teams",
  "description": "Search for soccer teams by name. Returns team details including ID, full name, country, and venue information.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "name": {
        "type": "string",
        "description": "Team name or partial name to search for (e.g., 'Charlotte FC', 'Real Madrid')"
      },
      "league": {
        "type": "string",
        "description": "Optional: Filter by league name or ID to narrow results",
        "optional": true
      },
      "country": {
        "type": "string", 
        "description": "Optional: Filter by country name or code",
        "optional": true
      }
    },
    "required": ["name"]
  }
}
```

## Expected Tool Response Format

```json
{
  "teams": [
    {
      "id": 1234,
      "name": "Charlotte FC",
      "code": "CHA",
      "country": "USA",
      "founded": 2019,
      "logo": "https://media.api-sports.io/football/teams/1234.png",
      "venue": {
        "name": "Bank of America Stadium",
        "city": "Charlotte",
        "capacity": 75523
      }
    }
  ],
  "total_results": 1,
  "search_term": "Charlotte FC"
}
```

## Rate Limiting & Caching Strategy

### Rate Limiting
- Track API calls per day/month
- Implement exponential backoff for rate limit errors
- Queue requests if approaching limits

### Caching Strategy
- Cache team search results for 24 hours
- Use team name hash as cache key
- Cache popular teams (top 100 worldwide) for longer periods
- Clear cache weekly or when new season starts

## Testing Strategy

### Unit Tests
- Test with exact team names
- Test with partial team names
- Test with common misspellings
- Test error conditions (invalid API key, network errors)

### Integration Tests
- Test against real API (with valid key)
- Verify response parsing
- Test rate limiting behavior

### Test Cases
```
1. Exact match: "Charlotte FC" → Should return Charlotte FC
2. Partial match: "Charlotte" → Should return Charlotte FC (and possibly others)
3. Common variations: "Man United" → Should return Manchester United
4. Non-existent team: "Fake Team FC" → Should return empty results
5. Ambiguous search: "United" → Should return multiple United teams
```

## Implementation Priority

1. **Phase 1**: Basic team search with exact name matching
2. **Phase 2**: Add fuzzy matching and variations handling  
3. **Phase 3**: Add caching and rate limiting
4. **Phase 4**: Add league/country filtering
5. **Phase 5**: Add error recovery and suggestions

## Security Considerations

- Store API key in environment variables
- Never log API keys or sensitive data
- Validate all user inputs
- Implement request timeout limits
- Rate limit user requests to prevent API abuse

## Performance Targets

- Response time: < 2 seconds for team search
- Cache hit rate: > 80% for popular teams
- API error rate: < 5%
- Memory usage: < 100MB for cache storage