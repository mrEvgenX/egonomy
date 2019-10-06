# Эгономика

Приложение для учета финансов.

Чтобы запустить нужно:
- Создать базу данных egonomic и пользователя egonomist
- Вручную применить миграции
- Выполнить C:\Go\bin\go.exe build
- Запустить egonomy.exe

## Локальные тесты
set DATABASE_URL=postgres://egonomist:1234@localhost:5432/egonomic
migrate -database $DATABASE_URL -path migrations up