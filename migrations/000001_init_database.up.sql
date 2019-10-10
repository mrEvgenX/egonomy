CREATE TABLE IF NOT EXISTS users(
	id serial PRIMARY KEY,
	email varchar(256) NOT NULL UNIQUE,
	sha bytea NOT NULL,
	salt varchar(128) NOT NULL
);

CREATE TABLE IF NOT EXISTS categories(
	id serial PRIMARY KEY,
	name varchar(64) NOT NULL,
	user_id integer
        REFERENCES users(id)
        ON DELETE CASCADE
        ON UPDATE CASCADE,
    UNIQUE(name, user_id)
);

CREATE TABLE IF NOT EXISTS transactions(
	id serial PRIMARY KEY,
    user_id integer
        REFERENCES users(id)
        ON DELETE SET NULL
        ON UPDATE CASCADE,
	"date" date NOT NULL,
	category integer 
        REFERENCES categories(id)
        ON DELETE SET NULL
        ON UPDATE CASCADE,
	amount real NOT NULL CHECK (amount > 0),
	comment varchar(256)
);
