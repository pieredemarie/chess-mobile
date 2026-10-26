ALTER TABLE users ADD COLUMN rating integer NOT NULL DEFAULT 1200 CHECK (rating >= 0);
ALTER TABLE games ADD COLUMN white_rating_before integer;
ALTER TABLE games ADD COLUMN black_rating_before integer;
ALTER TABLE games ADD COLUMN white_rating_after integer;
ALTER TABLE games ADD COLUMN black_rating_after integer;
CREATE INDEX games_white_history_idx ON games (white_id, created_at DESC, id DESC);
CREATE INDEX games_black_history_idx ON games (black_id, created_at DESC, id DESC);
