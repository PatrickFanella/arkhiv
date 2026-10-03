-- Run as the catalog owner. Provision the LOGIN password in private operator
-- input, and put its URL in PUBLIC_DATABASE_URL. This script contains no secret.
-- The role receives no write, migration, source snapshot or review-history access.
GRANT CONNECT ON DATABASE arkhiv TO arkhiv_public;
GRANT USAGE ON SCHEMA public TO arkhiv_public;
GRANT SELECT ON collections, documents, variants, assets, derivatives, page_text TO arkhiv_public;
ALTER ROLE arkhiv_public SET default_transaction_read_only = on;
ALTER ROLE arkhiv_public SET statement_timeout = '10s';
