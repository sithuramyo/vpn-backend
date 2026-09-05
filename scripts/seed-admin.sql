-- Provision the first administrator.
--
-- Admins are never auto-created from an arbitrary Google sign-in, so at
-- least one row must exist here before anyone can log in. Find your
-- Google "sub" by decoding the id_token JWT from a sign-in attempt (or via
-- https://developers.google.com/oauthplayground), then run:
--
--   psql "$DATABASE_URL" -v google_sub="'<sub>'" -v email="'<email>'" -v name="'<name>'" -f scripts/seed-admin.sql

INSERT INTO admins (google_sub, email, name, role, status)
VALUES (:google_sub, :email, :name, 'ADMIN', 'ACTIVE')
ON CONFLICT (google_sub) DO UPDATE SET
    role = 'ADMIN',
    status = 'ACTIVE';
