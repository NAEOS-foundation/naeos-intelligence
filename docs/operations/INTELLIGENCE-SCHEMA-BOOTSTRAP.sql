-- Reference only. Do not run directly against a live project.
-- Port this DDL into the canonical NAEOS Core Supabase migration workflow after review.
-- This creates NOLOGIN group roles and the isolated schema only. LOGIN credentials
-- must be provisioned separately through the approved secret-management process.

DO $bootstrap$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='naeos_intelligence_runtime') THEN
    CREATE ROLE naeos_intelligence_runtime NOLOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='naeos_intelligence_migrator') THEN
    CREATE ROLE naeos_intelligence_migrator NOLOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS;
  END IF;
END
$bootstrap$;

ALTER ROLE naeos_intelligence_runtime NOLOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS;
ALTER ROLE naeos_intelligence_migrator NOLOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS;

CREATE SCHEMA IF NOT EXISTS naeos_intelligence AUTHORIZATION naeos_intelligence_migrator;
ALTER SCHEMA naeos_intelligence OWNER TO naeos_intelligence_migrator;

GRANT USAGE ON SCHEMA naeos_intelligence TO naeos_intelligence_runtime;
GRANT CREATE ON SCHEMA naeos_intelligence TO naeos_intelligence_migrator;
REVOKE CREATE ON SCHEMA naeos_intelligence FROM naeos_intelligence_runtime;

-- Provision the following LOGIN roles separately with secret-managed passwords.
-- The runtime/migrator application code requires each LOGIN role to have exactly
-- one direct membership, respectively, in its matching NOLOGIN group role.
--
-- CREATE ROLE naeos_intelligence_runtime_login LOGIN INHERIT
--   NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS PASSWORD '<secret-managed>';
-- GRANT naeos_intelligence_runtime TO naeos_intelligence_runtime_login;
-- ALTER ROLE naeos_intelligence_runtime_login SET search_path TO 'pg_catalog, naeos_intelligence';
--
-- CREATE ROLE naeos_intelligence_migrator_login LOGIN INHERIT
--   NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS PASSWORD '<secret-managed>';
-- GRANT naeos_intelligence_migrator TO naeos_intelligence_migrator_login;
-- ALTER ROLE naeos_intelligence_migrator_login SET search_path TO 'pg_catalog, naeos_intelligence';

-- No default table grants are created here. Migration 00003 grants only the
-- active API call-path privileges to naeos_intelligence_runtime.
-- No CRM public, Control Plane, Identity, or service_role privileges are granted.
