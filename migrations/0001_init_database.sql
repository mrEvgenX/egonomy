CREATE TABLE users(
	id serial PRIMARY KEY,
	email varchar(256) NOT NULL UNIQUE,
	sha varchar(256) NOT NULL,
	salt varchar(128) NOT NULL,
	online boolean NOT NULL
);

CREATE TABLE categories(
	id serial PRIMARY KEY,
	name varchar(64) NOT NULL,
	user_id integer REFERENCES users(id)
);

CREATE TABLE transactions(
	id serial PRIMARY KEY,
	"date" date NOT NULL,
	category integer REFERENCES categories(id),
	amount real NOT NULL CHECK (amount > 0),
	comment varchar(256)
);
