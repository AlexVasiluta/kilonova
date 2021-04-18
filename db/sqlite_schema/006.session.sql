CREATE TABLE IF NOT EXISTS sessions (
	id 			TEXT 		PRIMARY KEY,
	created_at 	TIMESTAMP 	NOT NULL DEFAULT CURRENT_TIMESTAMP,
	user_id 	INTEGER		NOT NULL REFERENCES users(id)
);