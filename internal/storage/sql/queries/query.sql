-- name: CreateOrganization :one
INSERT INTO corti.organizations (organization_id, slug, region)
VALUES ($1, $2, $3)
RETURNING organization_id, slug, region, created_at;

-- name: OrganizationBySlug :one
SELECT
    organization_id,
    slug,
    region,
    created_at
FROM corti.organizations
WHERE slug = $1;
