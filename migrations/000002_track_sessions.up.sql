CREATE TABLE IF NOT EXISTS sessions(
    initiated timestamp with time zone NOT NULL,
    user_id integer NOT NULL
        REFERENCES users(id)
        ON DELETE CASCADE
        ON UPDATE CASCADE,
    ip varchar(128),
    user_agent varchar(256),
	token varchar(256) NOT NULL,
    PRIMARY KEY(user_id, token)
);
