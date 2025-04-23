# Digital Parenting App - Backend API (Child Reward System)

[![Go Version](https://img.shields.io/github/go-mod/go-version/rakaarfi/digital-parenting-app-be?style=flat-square)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-blue.svg?style=flat-square)](LICENSE)

Backend API for a Digital Parenting application enabling parents to assign tasks to children, award points upon completion, and allow children to redeem points for rewards.

## Table of Contents

- [Short Description](#short-description)
- [Key Features ✨](#key-features-)
- [Technology Stack](#technology-stack)
- [Prerequisites](#prerequisites)
- [Installation & Setup ⚙️](#installation--setup-️)
- [Running the Application ▶️](#running-the-application-)
- [API Documentation (Swagger)](#api-documentation-swagger)
- [API Endpoints Overview](#api-endpoints-overview)
- [Database Migrations](#database-migrations)
- [Project Structure (Overview)](#project-structure-overview)
- [Building the Application](#building-the-application)
- [Linting & Formatting](#linting--formatting)
- [Testing](#testing)
- [Contributing](#contributing)
- [License](#license)


## Short Description

This application facilitates a task-based reward system between parents and children. Parents can register, create accounts for their children, define tasks with associated points, and specify rewards that can be claimed. Children can view tasks, mark them as complete, check their points, and submit reward claims. Parents then verify tasks and approve/reject reward claims.

## Key Features ✨

*   **Authentication:** Registration (Parent/Admin), Login (All Roles), JWT (JSON Web Token) for sessions.
*   **Role Management:** Parent, Child, Admin (with potential Role CRUD by Admin).
*   **User Management:**
    *   Parent creates Child accounts.
    *   Admin manages all users (CRUD).
    *   Users manage their own profiles and passwords.
*   **Parent-Child Relationship:** Links parent and child accounts.
*   **Task Management:**
    *   Parent creates/manages Task definitions (templates).
    *   Parent assigns Tasks to Child.
    *   Child views and submits Tasks.
    *   Parent verifies (approve/reject) submitted Tasks.
*   **Reward Management:**
    *   Parent creates/manages Reward definitions.
    *   Child views available Rewards (from their parents).
    *   Child submits Reward claims.
    *   Parent reviews (approve/reject) Reward claims.
*   **Point System:**
    *   Points automatically added on Task approval.
    *   Points automatically deducted on Reward Claim approval.
    *   Parent (or Admin) can manually adjust points.
    *   Child can view point balance and transaction history.
*   **Authorization:** Role-based access control (Parent, Child, Admin) for endpoints.
*   **API Documentation:** Integrated Swagger UI.

## Technology Stack 

*   **Language:** Go (Golang)
*   **Web Framework:** Fiber v2
*   **Database:** PostgreSQL
*   **Database Driver:** pgx/v5 (pgxpool)
*   **Logging:** Zerolog
*   **Configuration:** Godotenv (for .env files)
*   **Authentication:** JWT (golang-jwt/v5)
*   **Password Hashing:** Bcrypt
*   **Validation:** Validator v10
*   **Database Migrations:** golang-migrate/migrate
*   **API Documentation:** Swaggo (swag)

## Prerequisites 

Before you begin, ensure your system has:

1.  **Go:** Version 1.18 or higher. [Install Go](https://go.dev/doc/install)
2.  **PostgreSQL:** A running PostgreSQL database server. [Install PostgreSQL](https://www.postgresql.org/download/)
3.  **`migrate` CLI:** CLI tool for running database migrations. [Install golang-migrate](https://github.com/golang-migrate/migrate/tree/master/cmd/migrate)
    ```bash
    # Example installation (check official docs for latest/other methods)
    go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
    ```
4.  **(Optional) Docker & Docker Compose:** For running PostgreSQL in a container.

## Installation & Setup ⚙️

1.  **Clone the Repository:**
    ```bash
    git clone https://github.com/rakaarfi/digital-parenting-app-be
    cd digital-parenting-app-be
    ```

2.  **Install Dependencies:**
    ```bash
    go mod tidy
    ```

3.  **Set Up Environment Variables:**
    *   Copy the `.env.example` file to `.env`:
        ```bash
        cp .env.example .env
        ```
    *   **Edit the `.env` file** and fill in all variables with appropriate values for your local environment. **IMPORTANT:** Ensure `JWT_SECRET` is set to a strong, random, and secret string (at least 32 characters recommended).

    *Example `.env` (Replace with your values):*
    ```dotenv
    # Application Configuration
    APP_PORT=3001

    # Database Configuration (PostgreSQL)
    DB_HOST=localhost
    DB_PORT=5432
    DB_USER=your_db_user
    DB_PASSWORD=your_db_password
    DB_NAME=digital_parenting_db
    DB_SSLMODE=disable # ('require', 'verify-full' for production with SSL)

    # JWT Configuration
    JWT_SECRET=your_very_strong_and_secret_jwt_key_at_least_32_chars # Change this!

    # Logger Configuration (Optional - Defaults provided in code)
    LOG_LEVEL=debug # trace, debug, info, warn, error, fatal, panic
    LOG_FORMAT=console # console (human-readable) or json
    LOG_FILE_ENABLED=false # true or false
    LOG_FILE_PATH=./logs/app.log
    LOG_FILE_MAX_SIZE_MB=100
    LOG_FILE_MAX_BACKUPS=5
    LOG_FILE_MAX_AGE_DAYS=30
    LOG_FILE_COMPRESS=false
    ```

4.  **Set Up the Database:**
    *   Ensure your PostgreSQL server is running.
    *   Create a new database in PostgreSQL with the name you defined in `DB_NAME` in your `.env` file. Example using `psql`:
        ```sql
        CREATE DATABASE digital_parenting_db;
        -- Optional: Create a dedicated user if you haven't already
        -- CREATE USER your_db_user WITH PASSWORD 'your_db_password';
        -- GRANT ALL PRIVILEGES ON DATABASE digital_parenting_db TO your_db_user;
        ```

5.  **Run Database Migrations:**
    *   Replace `<db_url>` below with your full PostgreSQL connection URL (based on your `.env` variables). The general format is: `postgres://<user>:<password>@<host>:<port>/<dbname>?sslmode=<sslmode>`
    *   Run the `migrate` command from the project's root directory:
        ```bash
        # Example URL: postgres://your_db_user:your_db_password@localhost:5432/digital_parenting_db?sslmode=disable
        migrate -database "postgres://your_db_user:your_db_password@localhost:5432/digital_parenting_db?sslmode=disable" -path migrations up
        ```
    *   This will create all necessary tables, types, functions, triggers, and indexes.

## Running the Application ▶️

Once the setup is complete, run the application using:

```bash
go run cmd/api/main.go
```

The server will start on the port specified in `APP_PORT` (default: 3001). You will see logs in the console (and in a file if enabled).

## API Documentation (Swagger) 

This application uses Swaggo to generate OpenAPI (Swagger) documentation.

1.  **Generate Documentation:** (Only needed if you modify GoDoc comments in handlers)
    ```bash
    swag init -g cmd/api/main.go
    ```
2.  **Access Documentation:** Once the server is running, open your browser and navigate to:
    `http://localhost:<APP_PORT>/swagger/index.html`
    (Example: `http://localhost:3001/swagger/index.html`)

    You can view all endpoints, parameters, request bodies, example responses, and try the API directly from the Swagger UI (remember to add your JWT token in the "Authorize" button for protected endpoints).

## API Endpoints Overview

Base Path: `/api/v1`

*   **Authentication (`/auth`)** [Public]
    *   `POST /register`: Register a new Parent or Admin account.
    *   `POST /login`: Login for Parent, Child, or Admin.
*   **Admin (`/admin`)** [Requires Admin Role]
    *   `GET /users`: Get all users (paginated).
    *   `GET /users/{userId}`: Get specific user details.
    *   `PATCH /users/{userId}`: Update user details.
    *   `DELETE /users/{userId}`: Delete a user (cannot delete self).
    *   `POST /roles`: Create a new role.
    *   `GET /roles`: Get all roles.
    *   `PATCH /roles/{roleId}`: Update a role.
    *   `DELETE /roles/{roleId}`: Delete a role (cannot delete base roles, fails if in use).
*   **User (`/user`)** [Requires Any Logged-in Role]
    *   `GET /profile`: Get own profile details.
    *   `PATCH /profile`: Update own profile details.
    *   `PATCH /password`: Change own password.
*   **Parent (`/parent`)** [Requires Parent Role]
    *   `POST /children/create`: Create a new child account and link it.
    *   `GET /children`: Get list of linked children.
    *   `DELETE /children/{childId}`: Remove link to a child.
    *   `POST /tasks`: Create a new task definition.
    *   `GET /tasks`: Get task definitions created by this parent (paginated).
    *   `PATCH /tasks/{taskId}`: Update own task definition.
    *   `DELETE /tasks/{taskId}`: Delete own task definition (fails if assigned).
    *   `POST /children/{childId}/tasks`: Assign a task definition to a specific child.
    *   `GET /children/{childId}/tasks`: Get tasks assigned to a specific child (filter by status).
    *   `PATCH /tasks/{userTaskId}/verify`: Verify (approve/reject) a child's submitted task.
    *   `POST /rewards`: Create a new reward definition.
    *   `GET /rewards`: Get reward definitions created by this parent (paginated).
    *   `PATCH /rewards/{rewardId}`: Update own reward definition.
    *   `DELETE /rewards/{rewardId}`: Delete own reward definition (fails if claimed).
    *   `GET /claims/pending`: Get pending reward claims from linked children (paginated).
    *   `PATCH /claims/{claimId}/review`: Review (approve/reject) a child's reward claim.
    *   `POST /children/{childId}/points`: Manually adjust points for a specific child.
*   **Child (`/child`)** [Requires Child Role]
    *   `GET /tasks`: Get own assigned tasks (filter by status, paginated).
    *   `PATCH /tasks/{userTaskId}/submit`: Submit a specific assigned task.
    *   `GET /points`: Get own current points balance.
    *   `GET /points/history`: Get own points transaction history (paginated).
    *   `GET /rewards`: Get available rewards from linked parents (paginated).
    *   `POST /rewards/{rewardId}/claim`: Claim a specific reward.
    *   `GET /claims`: Get own reward claim history (filter by status, paginated).
*   **Public (`/`)**
    *   `GET /health`: API health check.

## Database Migrations 

Database migrations are managed using `golang-migrate/migrate`. SQL migration files are located in the `migrations/` directory.

*   **Applying Migrations:**
    ```bash
    migrate -database "<db_url>" -path migrations up
    ```
*   **Rolling Back the Last Migration:**
    ```bash
    migrate -database "<db_url>" -path migrations down 1
    ```
*   **Rolling Back All Migrations:**
    ```bash
    migrate -database "<db_url>" -path migrations down
    ```
    *(Replace `<db_url>` with your database connection URL)*

## Project Structure (Overview) 

```
.
├── cmd/api/main.go         # Application entry point
├── configs/                # Configuration loading (env vars)
├── docs/                   # Generated Swagger documentation files
├── internal/               # Internal application code (not exported)
│   ├── api/v1/             # API v1 handlers and routing
│   │   ├── handlers/       # HTTP handler logic (Admin, Auth, Child, Parent, User, Error)
│   │   └── routes.go       # API v1 route definitions
│   ├── database/           # Database connection (PostgreSQL)
│   ├── logger/             # Logger setup (Zerolog)
│   ├── middleware/         # Fiber middleware (Auth, Global, etc.)
│   ├── models/             # Data struct definitions and input DTOs
│   ├── repository/         # Data Access Layer (Interfaces & Repo Implementations)
│   ├── service/            # Business Logic Layer (Interfaces & Service Implementations)
│   └── utils/              # Utility functions (Hash, JWT, Pagination, Validation)
├── logs/                   # Directory for log files (if enabled)
├── migrations/             # SQL database migration files (.up.sql, .down.sql)
├── go.mod                  # Go dependency management
├── go.sum                  # Dependency checksums
├── .env.example            # Example environment variable file
├── .gitignore              # Files/folders ignored by Git
└── README.md               # This file
```

## Building the Application

To build a standalone binary executable for Linux (suitable for containers):

```bash
CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o digital-parenting-be cmd/api/main.go
```

*   `CGO_ENABLED=0`: Disables CGO for static linking.
*   `GOOS=linux`: Specifies the target operating system.
*   `-ldflags="-w -s"`: Reduces binary size by stripping debug information and symbols.
*   `-o digital-parenting-be`: Specifies the output filename.

This will create an executable file named `digital-parenting-be` in the project root. For other operating systems, adjust `GOOS` (e.g., `darwin` for macOS, `windows` for Windows).

## Linting & Formatting

Consistent code style and quality are maintained using standard Go tools and `golangci-lint`.

*   **Formatting:** Ensure your code is formatted according to Go standards.
    ```bash
    go fmt ./...
    # OR
    gofmt -w .
    ```
*   **Linting:** Use `golangci-lint` to run a suite of linters.
    1.  **Install `golangci-lint`** (if not already installed): [Installation Guide](https://golangci-lint.run/usage/install/)
        ```bash
        # Example installation
        go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
        ```
    2.  **Run Linters:**
        ```bash
        golangci-lint run
        ```
        This will use the configuration file `.golangci.yml` (if present) or default settings. Consider creating a `.golangci.yml` for project-specific rules.


<!-- ## Testing -->

<!-- *(TODO: Add instructions on how to run unit or integration tests if you implement them)* -->

## Contributing 

Contributions are welcome! Please create an Issue or Pull Request.

## License 

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
