CREATE TABLE IF NOT EXISTS leagues (
  id INTEGER PRIMARY KEY,
  name TEXT NOT NULL,
  hidden BOOLEAN DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS fixtures (
  id INTEGER PRIMARY KEY,
  league_id INTEGER,
  season INTEGER,
  home_id INTEGER,
  away_id INTEGER,
  timestamp INTEGER,
  finished BOOLEAN,
  home_goals INTEGER,
  away_goals INTEGER,
  FOREIGN KEY (league_id) REFERENCES leagues(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_fixtures_league_season ON fixtures(league_id, season);
CREATE INDEX IF NOT EXISTS idx_fixtures_home_team ON fixtures(home_id, league_id, season);
CREATE INDEX IF NOT EXISTS idx_fixtures_away_team ON fixtures(away_id, league_id, season);
CREATE INDEX IF NOT EXISTS idx_fixtures_finished ON fixtures(finished);

CREATE TABLE IF NOT EXISTS fixture_stats (
  fixture_id INTEGER,
  team_id INTEGER,
  league_id INTEGER,
  season INTEGER,
  shots INTEGER,
  shots_on_goal INTEGER,
  shots_blocked INTEGER,
  shots_in_box INTEGER,
  shots_out_box INTEGER,
  possession REAL,
  passes INTEGER,
  passes_completed INTEGER,
  fouls INTEGER,
  corners INTEGER,
  offsides INTEGER,
  yellow_cards INTEGER,
  red_cards INTEGER,
  xg REAL,
  goals_prevented INTEGER,
  PRIMARY KEY (fixture_id, team_id),
  FOREIGN KEY (fixture_id) REFERENCES fixtures(id) ON DELETE CASCADE,
  FOREIGN KEY (league_id) REFERENCES leagues(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_fixture_stats_team_league_season ON fixture_stats(team_id, league_id, season);
CREATE INDEX IF NOT EXISTS idx_fixture_stats_league_season ON fixture_stats(league_id, season);

CREATE TABLE IF NOT EXISTS teams (
  id INTEGER PRIMARY KEY,
  name TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS bets (
  id INTEGER PRIMARY KEY,
  match_id INTEGER,
  name TEXT,
  amount REAL,
  line REAL,
  odds INTEGER,
  result TEXT
);

CREATE TABLE IF NOT EXISTS juice (
  id TEXT PRIMARY KEY,
  raw_data TEXT
);
