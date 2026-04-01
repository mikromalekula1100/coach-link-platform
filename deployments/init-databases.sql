-- auth_db is created by POSTGRES_DB env var, only create the rest
SELECT 'CREATE DATABASE user_db' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'user_db')\gexec
SELECT 'CREATE DATABASE training_db' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'training_db')\gexec
SELECT 'CREATE DATABASE notification_db' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'notification_db')\gexec
