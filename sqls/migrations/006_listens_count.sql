-- +migrate Up
alter table tracks add column listens_count integer;

-- +migrate Down
alter table tracks drop column listens_count;