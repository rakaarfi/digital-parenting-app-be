-- Migrations Down

-- Hapus Trigger DULU
DROP TRIGGER IF EXISTS set_timestamp_users ON users;
DROP TRIGGER IF EXISTS set_timestamp_roles ON roles;
DROP TRIGGER IF EXISTS set_timestamp_user_relationship ON user_relationship;
DROP TRIGGER IF EXISTS set_timestamp_tasks ON tasks;
DROP TRIGGER IF EXISTS set_timestamp_user_tasks ON user_tasks;
DROP TRIGGER IF EXISTS set_timestamp_rewards ON rewards; -- Trigger rewards ada di up.sql tapi tidak ada di down.sql awal Anda? Pastikan ada.
DROP TRIGGER IF EXISTS set_timestamp_user_rewards ON user_rewards;
DROP TRIGGER IF EXISTS set_timestamp_point_transactions ON point_transactions;

-- BARU Hapus Function
DROP FUNCTION IF EXISTS trigger_set_timestamp();

-- Hapus Index
DROP INDEX IF EXISTS idx_user_relationship_parent_id;
DROP INDEX IF EXISTS idx_user_relationship_child_id;
DROP INDEX IF EXISTS idx_user_tasks_user_id;
DROP INDEX IF EXISTS idx_user_tasks_task_id;
DROP INDEX IF EXISTS idx_user_rewards_user_id;
DROP INDEX IF EXISTS idx_user_rewards_reward_id;
DROP INDEX IF EXISTS idx_point_transactions_user_id;
DROP INDEX IF EXISTS idx_point_transactions_related_user_task_id;
DROP INDEX IF EXISTS idx_point_transactions_related_user_reward_id;
DROP INDEX IF EXISTS idx_point_transactions_created_by_user_id;

-- Hapus Custom Types (ENUM)
DROP TYPE IF EXISTS user_task_status;
DROP TYPE IF EXISTS user_reward_status;
DROP TYPE IF EXISTS point_transaction_type;

-- Hapus Tabel (urutan terbalik dari pembuatan/dependensi FK)
DROP TABLE IF EXISTS point_transactions;
DROP TABLE IF EXISTS user_rewards;
DROP TABLE IF EXISTS user_tasks;
DROP TABLE IF EXISTS user_relationship;
DROP TABLE IF EXISTS tasks;
DROP TABLE IF EXISTS rewards;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS roles;