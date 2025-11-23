-- +migrate Up

create extension if not exists "hstore";

create table blobs
(
    id         uuid        not null,
    created_at timestamptz not null,
    updated_at timestamptz not null,

    digest     text        not null,
    size       bigint      not null,

    primary key (id),
    unique (digest)
);

create table files
(
    id          uuid        not null,
    created_at  timestamptz not null,
    updated_at  timestamptz not null,

    digest      text        not null,
    contentType text        not null,
    data        bytea       not null,
    size        bigint,

    primary key (id)
);

create table tenants
(
    id                     uuid        not null,
    created_at             timestamptz not null,
    updated_at             timestamptz not null,

    slug                   text        not null,
    display_name            text        not null,

    oidc_client            text        not null,
    oidc_issuer            text        not null,
    oidc_role_claim        text        not null,
    oidc_role_claim_format text        not null,
    oidc_role_mapping      hstore      not null,

    primary key (id),
    unique (slug)
);

create table projects
(
    id          uuid        not null,
    created_at  timestamptz not null,
    updated_at  timestamptz not null,

    tenant_id   uuid        not null,

    slug        text        not null,
    display_name text        not null,
    description text,

    primary key (id),
    foreign key (tenant_id) references tenants (id),
    unique (tenant_id, slug)
);

create table repositories
(
    id             uuid        not null,
    created_at     timestamptz not null,
    updated_at     timestamptz not null,

    project_id     uuid        not null,

    slug           text        not null,
    display_name    text        not null,
    description    text,
    readme_file_id uuid,

    is_public      boolean     not null,

    primary key (id),
    foreign key (project_id) references projects (id),
    foreign key (readme_file_id) references files (id),
    unique (project_id, slug)
);

create table repository_blobs
(
    id            uuid        not null,
    created_at    timestamptz not null,
    updated_at    timestamptz not null,

    repository_id uuid        not null,
    blob_id       uuid        not null,

    primary key (id),
    foreign key (repository_id) references repositories (id),
    foreign key (blob_id) references blobs (id),
    unique (repository_id, blob_id)
);

create table manifests
(
    id            uuid        not null,
    created_at    timestamptz not null,
    updated_at    timestamptz not null,

    repository_id uuid        not null,
    blob_id       uuid        not null,

    digest        text        not null,

    primary key (id),
    foreign key (repository_id) references repositories (id),
    foreign key (blob_id) references blobs (id),
    unique (repository_id, digest)
);

create table tags
(
    id            uuid        not null,
    created_at    timestamptz not null,
    updated_at    timestamptz not null,

    repository_id uuid        not null,
    manifest_id   uuid        not null,

    name          text        not null,

    primary key (id),
    foreign key (repository_id) references repositories (id),
    foreign key (manifest_id) references manifests (id),
    unique (repository_id, name)
);

create table users
(
    id           uuid        not null,
    created_at   timestamptz not null,
    updated_at   timestamptz not null,

    tenant_id    uuid        not null,

    oidc_subject text        not null,
    display_name text,
    email        text,

    primary key (id),
    foreign key (tenant_id) references tenants (id),
    unique (tenant_id, oidc_subject)
);

create table pats
(
    id            uuid        not null,
    created_at    timestamptz not null,
    updated_at    timestamptz not null,

    user_id       uuid        not null,
    display_name  text        not null,
    hashed_secret bytea       not null,

    primary key (id),
    foreign key (user_id) references users (id)
);

create table project_accesses
(
    id         uuid        not null,
    created_at timestamptz not null,
    updated_at timestamptz not null,

    project_id uuid        not null,
    user_id    uuid        not null,
    "role"     text        not null,

    primary key (id),
    foreign key (project_id) references projects (id),
    foreign key (user_id) references users (id),
    unique (project_id, user_id)
);


create table repository_accesses
(
    id            uuid        not null,
    created_at    timestamptz not null,
    updated_at    timestamptz not null,

    repository_id uuid        not null,
    user_id       uuid        not null,
    "role"        text        not null,

    primary key (id),
    foreign key (repository_id) references repositories (id),
    foreign key (user_id) references users (id),
    unique (repository_id, user_id)
);

-- +migrate Down
