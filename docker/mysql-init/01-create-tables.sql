CREATE DATABASE IF NOT EXISTS migrations;
USE migrations;

CREATE TABLE IF NOT EXISTS tasks (
  id VARCHAR(36) PRIMARY KEY,
  source VARCHAR(100) NOT NULL,
  target VARCHAR(100) NOT NULL,
  payload TEXT,
  status VARCHAR(20) NOT NULL,
  result TEXT,
  error TEXT,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL
);

-- helpful indexes for queries (MySQL 8+ supports IF NOT EXISTS on CREATE INDEX)
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks (status);
CREATE INDEX IF NOT EXISTS idx_tasks_target ON tasks (target);

-- identities table for mapping Zoom users to Teams users
CREATE TABLE IF NOT EXISTS identities (
  zoom_user_id VARCHAR(100) PRIMARY KEY,
  zoom_user_email VARCHAR(255),
  zoom_user_display_name VARCHAR(255),
  teams_user_id VARCHAR(100),
  teams_user_principal_name VARCHAR(255),
  teams_user_display_name VARCHAR(255),
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL
);

-- ensure uniqueness for Teams user id mapping
CREATE UNIQUE INDEX IF NOT EXISTS idx_teams_id ON identities (teams_user_id);
