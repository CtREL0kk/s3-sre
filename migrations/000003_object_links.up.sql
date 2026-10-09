CREATE TABLE object_links (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id   uuid NOT NULL REFERENCES objects (id) ON DELETE CASCADE,
    target_id   uuid REFERENCES objects (id) ON DELETE CASCADE,
    target_name text NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX object_links_source_idx ON object_links (source_id);
CREATE INDEX object_links_target_idx ON object_links (target_id);