DROP TABLE IF EXISTS accounting_books;
DROP TABLE IF EXISTS user_book_link;
DROP TABLE IF EXISTS transaction_book_link;
ALTER TABLE transactions DROP COLUMN book_id;
ALTER TABLE users DROP COLUMN default_accounting_book;