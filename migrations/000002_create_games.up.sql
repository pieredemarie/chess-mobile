CREATE TABLE games (
    id uuid PRIMARY KEY,
    white_id uuid NOT NULL REFERENCES users(id),
    black_id uuid NOT NULL REFERENCES users(id),
    fen text NOT NULL,
    ply integer NOT NULL DEFAULT 0 CHECK (ply >= 0),
    result text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT games_distinct_players CHECK (white_id <> black_id),
    CONSTRAINT games_fen_nonempty CHECK (fen <> ''),
    CONSTRAINT games_result_valid CHECK (result IS NULL OR result IN ('1-0', '0-1', '1/2-1/2'))
);

CREATE TABLE game_moves (
    game_id uuid NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    ply integer NOT NULL CHECK (ply > 0),
    uci text NOT NULL,
    fen text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (game_id, ply)
);


