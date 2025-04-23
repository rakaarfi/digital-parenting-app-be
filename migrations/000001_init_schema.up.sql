-- Migrations Up

-- Tabel Peran
CREATE TABLE roles (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Tabel Pengguna
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(100) NOT NULL UNIQUE,
    password VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    role_id INT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE RESTRICT
);

-- Tabel Relasi Parent dan Child
CREATE TABLE user_relationship (
    id SERIAL PRIMARY KEY,
    parent_id INT NOT NULL,
    child_id INT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (parent_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (child_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT unique_parent_child UNIQUE (parent_id, child_id)
);

-- Tabel Task
CREATE TABLE tasks (
    id SERIAL PRIMARY KEY,
    task_name VARCHAR(255) NOT NULL,
    task_point INT NOT NULL,
    task_description TEXT,
    created_by_user_id INT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (created_by_user_id) REFERENCES users(id) ON DELETE RESTRICT
);

-- Tabel Relasi User dan Task
CREATE TYPE user_task_status AS ENUM ('assigned', 'submitted', 'approved', 'rejected');
CREATE TABLE user_tasks (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL,
    task_id INT NOT NULL,
    assigned_by_user_id INT NOT NULL,
    status user_task_status NOT NULL,
    assigned_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    submitted_at TIMESTAMPTZ,
    verified_by_user_id INT,
    verified_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE RESTRICT,
    FOREIGN KEY (assigned_by_user_id) REFERENCES users(id) ON DELETE SET NULL,
    FOREIGN KEY (verified_by_user_id) REFERENCES users(id) ON DELETE SET NULL
);

-- Tabel Reward
CREATE TABLE rewards (
    id SERIAL PRIMARY KEY,
    reward_name VARCHAR(255) NOT NULL,
    reward_point INT NOT NULL,
    reward_description TEXT,
    created_by_user_id INT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (created_by_user_id) REFERENCES users(id) ON DELETE RESTRICT
);

-- Tabel Relasi User dan Reward
CREATE TYPE user_reward_status AS ENUM ('pending', 'approved', 'rejected');
CREATE TABLE user_rewards (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL,
    reward_id INT NOT NULL,
    points_deducted INT NOT NULL,
    claimed_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    status user_reward_status NOT NULL,
    reviewed_by_user_id INT,
    reviewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (reward_id) REFERENCES rewards(id) ON DELETE RESTRICT,
    FOREIGN KEY (reviewed_by_user_id) REFERENCES users(id) ON DELETE SET NULL
);

-- Tabel Point Transaction
CREATE TYPE point_transaction_type AS ENUM ('task_completion', 'reward_redemption', 'manual_adjustment');
CREATE TABLE point_transactions (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL,
    change_amount INT NOT NULL,
    transaction_type point_transaction_type NOT NULL,
    related_user_task_id INT,
    related_user_reward_id INT,
    created_by_user_id INT NOT NULL,
    notes TEXT,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (related_user_task_id) REFERENCES user_tasks(id) ON DELETE SET NULL,
    FOREIGN KEY (related_user_reward_id) REFERENCES user_rewards(id) ON DELETE SET NULL,
    FOREIGN KEY (created_by_user_id) REFERENCES users(id) ON DELETE SET NULL
);

-- Function to update updated_at column automatically
CREATE OR REPLACE FUNCTION trigger_set_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger for users table
CREATE TRIGGER set_timestamp_users
BEFORE UPDATE ON users
FOR EACH ROW
EXECUTE FUNCTION trigger_set_timestamp();

-- Trigger for roles table
CREATE TRIGGER set_timestamp_roles
BEFORE UPDATE ON roles
FOR EACH ROW
EXECUTE FUNCTION trigger_set_timestamp();

-- Trigger for user_relationship table
CREATE TRIGGER set_timestamp_user_relationship
BEFORE UPDATE ON user_relationship
FOR EACH ROW
EXECUTE FUNCTION trigger_set_timestamp();

-- Trigger for tasks table
CREATE TRIGGER set_timestamp_tasks
BEFORE UPDATE ON tasks
FOR EACH ROW
EXECUTE FUNCTION trigger_set_timestamp();

-- Trigger for user_tasks table
CREATE TRIGGER set_timestamp_user_tasks
BEFORE UPDATE ON user_tasks
FOR EACH ROW
EXECUTE FUNCTION trigger_set_timestamp();

-- Trigger for rewards table
CREATE TRIGGER set_timestamp_rewards
BEFORE UPDATE ON rewards
FOR EACH ROW
EXECUTE FUNCTION trigger_set_timestamp();

-- Trigger for user_rewards table
CREATE TRIGGER set_timestamp_user_rewards
BEFORE UPDATE ON user_rewards
FOR EACH ROW
EXECUTE FUNCTION trigger_set_timestamp();

-- Trigger for point_transactions table
CREATE TRIGGER set_timestamp_point_transactions
BEFORE UPDATE ON point_transactions
FOR EACH ROW
EXECUTE FUNCTION trigger_set_timestamp();

-- Tambahkan index untuk performa
CREATE INDEX idx_user_relationship_parent_id ON user_relationship (parent_id);
CREATE INDEX idx_user_relationship_child_id ON user_relationship (child_id);
CREATE INDEX idx_user_tasks_user_id ON user_tasks (user_id);
CREATE INDEX idx_user_tasks_task_id ON user_tasks (task_id);
CREATE INDEX idx_user_rewards_user_id ON user_rewards (user_id);
CREATE INDEX idx_user_rewards_reward_id ON user_rewards (reward_id);
CREATE INDEX idx_point_transactions_user_id ON point_transactions (user_id);
CREATE INDEX idx_point_transactions_related_user_task_id ON point_transactions (related_user_task_id);
CREATE INDEX idx_point_transactions_related_user_reward_id ON point_transactions (related_user_reward_id);
CREATE INDEX idx_point_transactions_created_by_user_id ON point_transactions (created_by_user_id);

-- Seed data Roles (contoh)
INSERT INTO roles (name) VALUES ('Parent'), ('Child'), ('Admin');