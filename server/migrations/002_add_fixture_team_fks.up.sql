PRAGMA foreign_keys=OFF;

CREATE TABLE fixtures_new (
  id INTEGER PRIMARY KEY,
  league_id INTEGER,
  season INTEGER,
  home_id INTEGER,
  away_id INTEGER,
  timestamp INTEGER,
  finished BOOLEAN,
  home_goals INTEGER,
  away_goals INTEGER,
  FOREIGN KEY (league_id) REFERENCES leagues(id) ON DELETE CASCADE,
  FOREIGN KEY (home_id) REFERENCES teams(id) ON DELETE CASCADE,
  FOREIGN KEY (away_id) REFERENCES teams(id) ON DELETE CASCADE
);

INSERT INTO fixtures_new (id, league_id, season, home_id, away_id, timestamp, finished, home_goals, away_goals)
  SELECT id, league_id, season, home_id, away_id, timestamp, finished, home_goals, away_goals
  FROM fixtures;

DROP TABLE fixtures;
ALTER TABLE fixtures_new RENAME TO fixtures;

CREATE INDEX idx_fixtures_league_season ON fixtures(league_id, season);
CREATE INDEX idx_fixtures_home_team ON fixtures(home_id, league_id, season);
CREATE INDEX idx_fixtures_away_team ON fixtures(away_id, league_id, season);
CREATE INDEX idx_fixtures_finished ON fixtures(finished);

PRAGMA foreign_keys=ON;
