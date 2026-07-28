CREATE TABLE observations (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  observed_at TEXT NOT NULL,
  status_json BLOB NOT NULL
);
INSERT INTO observations (observed_at, status_json) VALUES
  ('2026-07-27T10:00:00Z', '{"observed_at":"2026-07-27T10:00:00Z","connected":true,"power":true,"filter":true,"heater":false,"jets":false,"bubbles":false,"sanitizer":false,"unit":"C","preset_temp":36}');

CREATE TABLE events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  created_at TEXT NOT NULL,
  type TEXT NOT NULL,
  message TEXT NOT NULL,
  data_json BLOB
);
INSERT INTO events (created_at, type, message, data_json) VALUES
  ('2026-07-27T10:01:00Z', 'control_mode', 'control mode updated', CAST('{"manual_control":true}' AS BLOB));

CREATE TABLE commands (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  issued_at TEXT NOT NULL,
  completed_at TEXT,
  capability TEXT NOT NULL,
  state_json BLOB,
  value_json BLOB,
  source TEXT,
  success INTEGER NOT NULL DEFAULT 0,
  error TEXT,
  status_json BLOB
);
INSERT INTO commands (issued_at, completed_at, capability, state_json, source, success) VALUES
  ('2026-07-27T10:02:00Z', '2026-07-27T10:02:01Z', 'heater', 'true', 'webui-manual', 1);

CREATE TABLE kv (
  key TEXT PRIMARY KEY,
  value BLOB NOT NULL,
  updated_at TEXT NOT NULL
);
INSERT INTO kv (key, value, updated_at) VALUES
  ('control_mode', '{"manual_control":true}', '2026-07-27T10:00:00Z'),
  ('desired_state', '{"target_temp":36}', '2026-07-27T10:00:00Z');

CREATE TABLE plans (
  id TEXT PRIMARY KEY,
  updated_at TEXT NOT NULL,
  plan_json BLOB NOT NULL
);
INSERT INTO plans (id, updated_at, plan_json) VALUES
  ('daily-filter', '2026-07-27T10:00:00Z', '{"id":"daily-filter","type":"time_window","enabled":true,"capability":"filter","from":"02:00","to":"04:00"}'),
  ('legacy-kind', '2026-07-27T10:00:00Z', '{"id":"legacy-kind","kind":"manual_override","enabled":true}'),
  ('legacy-source', '2026-07-27T10:00:00Z', '{"id":"legacy-source","type":"time_window","source":"webui-pause","enabled":true,"capability":"filter","from":"00:00","to":"01:00"}'),
  ('webui-manual', '2026-07-27T10:00:00Z', '{"id":"webui-manual","type":"manual_override","enabled":true}');
