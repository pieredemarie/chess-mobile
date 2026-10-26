DROP INDEX games_black_history_idx;
DROP INDEX games_white_history_idx;
ALTER TABLE games DROP COLUMN black_rating_after;
ALTER TABLE games DROP COLUMN white_rating_after;
ALTER TABLE games DROP COLUMN black_rating_before;
ALTER TABLE games DROP COLUMN white_rating_before;
ALTER TABLE users DROP COLUMN rating;
