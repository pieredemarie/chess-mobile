CREATE TABLE users (
 id uuid PRIMARY KEY,
 nickname text NOT NULL,
 email text NOT NULL UNIQUE,
 password_hash text NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now()
);
