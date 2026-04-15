-- +migrate Up
alter table manifests add column media_type text not null default '';

-- +migrate Down
alter table manifests drop column media_type;
