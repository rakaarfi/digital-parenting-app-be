-- Migrations Down

-- Hapus Trigger DULU
DROP TRIGGER IF EXISTS set_timestamp_users ON users;
DROP TRIGGER IF EXISTS set_timestamp_roles ON roles;
DROP TRIGGER IF EXISTS set_timestamp_user_relationship ON user_relationship;
DROP TRIGGER IF EXISTS set_timestamp_tasks ON tasks;
DROP TRIGGER IF EXISTS set_timestamp_user_tasks ON user_tasks;
DROP TRIGGER IF EXISTS set_timestamp_rewards ON rewards;
DROP TRIGGER IF EXISTS set_timestamp_user_rewards ON user_rewards;
DROP TRIGGER IF EXISTS set_timestamp_point_transactions ON point_transactions;

-- Hapus Function SETELAH Trigger
DROP FUNCTION IF EXISTS trigger_set_timestamp();

-- Hapus Index (bisa sebelum atau sesudah function)
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

-- Hapus Tabel (urutan terbalik dari pembuatan/dependensi FK)
-- Tabel yang mereferensi dihapus dulu
DROP TABLE IF EXISTS point_transactions;
DROP TABLE IF EXISTS user_rewards;
DROP TABLE IF EXISTS user_tasks;
-- Tabel invitation_codes dihapus di migrasi down ke-2
DROP TABLE IF EXISTS user_relationship;
DROP TABLE IF EXISTS tasks;
DROP TABLE IF EXISTS rewards;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS roles;

-- BARU Hapus Custom Types (ENUM) SETELAH Tabel Dihapus
DROP TYPE IF EXISTS user_task_status;
DROP TYPE IF EXISTS user_reward_status;
DROP TYPE IF EXISTS point_transaction_type;