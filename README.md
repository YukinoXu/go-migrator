# go-migrator

A minimal Go project demonstrating a task manager for migration jobs.

Features

- Submit migration tasks via HTTP API
- Background worker(s) consume tasks from a queue and execute migrations
- Task status query via HTTP API
- Example mock migrator: Zoom chat -> Teams

Quick start

1. Build and run:

    ```powershell
    go run ./cmd/migrator
    ```

2. Submit a task:

    ```powershell
    curl -X POST http://localhost:8080/tasks -H "Content-Type: application/json" -d '{"source":"zoom","target":"teams","payload":{"conversation_id":"zoom-room-123"}}'
    ```

3. Query a task:

    ```powershell
    curl http://localhost:8080/tasks/<task-id>
    ```

Using MySQL for persistence

1. Start a MySQL server and create a database (example uses `migrations`):

    ```sql
    CREATE DATABASE migrations;
    ```

2. Run the migrator with the `MYSQL_DSN` environment variable. Example DSN:

    ```text
    user:password@tcp(127.0.0.1:3306)/migrations?parseTime=true
    ```

    On Windows PowerShell the run command looks like:

    ```powershell
    $env:MYSQL_DSN = 'user:password@tcp(127.0.0.1:3306)/migrations?parseTime=true'
    go run ./cmd/migrator
    ```

    When `MYSQL_DSN` is set the service will use MySQL for persistent tasks and bootstrap the required schema automatically.

Docker Compose

You can run a local MySQL and the migrator together using Docker Compose (provided in `docker-compose.yml`). Example:

```powershell
# build images and start services in the foreground
docker compose up --build

# or start in the background
docker compose up --build -d
```

The compose file exposes two host ports by default:

- `8080` → migrator HTTP API
- `3306` → MySQL

Port conflicts

- If you already run MySQL (or another service) on host port `3306`, change the host side mapping in `docker-compose.yml` (for example `"3307:3306"`) or stop the local service before starting the compose stack.
- If port `8080` is taken, change the mapping under the `migrator` service (for example `"8081:8080"`) and then use the new host port when calling the API.

Stopping and removing containers and volumes (careful: this removes DB data):

```powershell
docker compose down -v
```

This compose setup includes an init SQL script in `docker/mysql-init/` that bootstraps the `migrations` database and `tasks` table on first startup.
