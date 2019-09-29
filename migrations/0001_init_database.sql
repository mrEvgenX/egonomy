DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS categories CASCADE;
DROP TABLE IF EXISTS transactions CASCADE;

CREATE TABLE users(
	id serial PRIMARY KEY,
	email varchar(256) NOT NULL UNIQUE,
	sha bytea NOT NULL,
	salt varchar(128) NOT NULL,
	online boolean NOT NULL
);

CREATE TABLE categories(
	id serial PRIMARY KEY,
	name varchar(64) NOT NULL,
	user_id integer REFERENCES users(id),
    UNIQUE(name, user_id)
);

CREATE TABLE transactions(
	id serial PRIMARY KEY,
    user_id integer REFERENCES users(id),
	"date" date NOT NULL,
	category integer REFERENCES categories(id),
	amount real NOT NULL CHECK (amount > 0),
	comment varchar(256)
);
