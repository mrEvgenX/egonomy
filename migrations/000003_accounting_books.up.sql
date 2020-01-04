CREATE TABLE IF NOT EXISTS accounting_books(
	id serial PRIMARY KEY,
	name varchar(256) NOT NULL,
    created_by integer
        REFERENCES users(id)
        ON DELETE CASCADE
        ON UPDATE CASCADE,
    archived boolean NOT NULL DEFAULT False
);

CREATE TABLE IF NOT EXISTS user_book_link(
	user_id integer
        REFERENCES users(id)
        ON DELETE CASCADE
        ON UPDATE CASCADE,
    book_id integer
        REFERENCES accounting_books(id)
        ON DELETE CASCADE
        ON UPDATE CASCADE,
    UNIQUE(user_id, book_id)
);

ALTER TABLE transactions ADD COLUMN book_id integer NOT NULL
    REFERENCES accounting_books(id)
    ON DELETE CASCADE
    ON UPDATE CASCADE;

ALTER TABLE users ADD COLUMN default_accounting_book integer NOT NULL
    REFERENCES accounting_books(id)
    ON DELETE CASCADE
    ON UPDATE CASCADE;
-- Нужен ник для пользователя все-таки
-- У каждой книги будет админ, тот, кто ее создал, тот и молодец, хотя бы на начальном этапе
-- Если пользователь записал транзакцию в книгу, а его потом удалили, надо тразакцию продублировать (а когда его вернут - как-то понять, что это дубликат оказался)
