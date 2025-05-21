include .env
export

# Directory for migration files
MIGRATIONS_DIR := db/migrations

# Create a new migration (example: make create name=create_users_table)
create:
	goose -dir $(MIGRATIONS_DIR) create $(name) sql

# Run all up migrations
migrate:
	goose -dir $(MIGRATIONS_DIR) postgres "$(DB_URL)" up

# Roll back the last migration
rollback:
	goose -dir $(MIGRATIONS_DIR) postgres "$(DB_URL)" down

# Redo the last migration (down + up)
redo:
	goose -dir $(MIGRATIONS_DIR) postgres "$(DB_URL)" redo

# Show current status
status:
	goose -dir $(MIGRATIONS_DIR) postgres "$(DB_URL)" status

# Create migrations dir if it doesn't exist
init:
	mkdir -p $(MIGRATIONS_DIR)
