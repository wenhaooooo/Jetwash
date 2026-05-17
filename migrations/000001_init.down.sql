DROP TABLE IF EXISTS detection_match;
DROP TABLE IF EXISTS detection_history;
DROP TABLE IF EXISTS import_tasks;
DROP TABLE IF EXISTS sensitive_words;
DROP TABLE IF EXISTS api_keys;
DROP TABLE IF EXISTS tenants;
DROP EXTENSION IF EXISTS "uuid-ossp";
-- Note: Don't drop vector extension as other databases might use it
