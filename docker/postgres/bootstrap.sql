-- Local development only. These credentials are intentionally non-production.
-- The application and migration containers use separate LOGIN roles; the postgres
-- superuser is used only by this one-shot bootstrap service.
DO $bootstrap$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='naeos_intelligence_runtime') THEN
    CREATE ROLE naeos_intelligence_runtime NOLOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='naeos_intelligence_migrator') THEN
    CREATE ROLE naeos_intelligence_migrator NOLOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='naeos_intelligence_runtime_login') THEN
    CREATE ROLE naeos_intelligence_runtime_login LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='naeos_intelligence_migrator_login') THEN
    CREATE ROLE naeos_intelligence_migrator_login LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS;
  END IF;
END
$bootstrap$;

ALTER ROLE naeos_intelligence_runtime NOLOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS;
ALTER ROLE naeos_intelligence_migrator NOLOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS;
ALTER ROLE naeos_intelligence_runtime_login LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS PASSWORD 'naeos-runtime-dev';
ALTER ROLE naeos_intelligence_migrator_login LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS PASSWORD 'naeos-migrator-dev';

GRANT naeos_intelligence_runtime TO naeos_intelligence_runtime_login;
GRANT naeos_intelligence_migrator TO naeos_intelligence_migrator_login;

CREATE SCHEMA IF NOT EXISTS naeos_intelligence AUTHORIZATION naeos_intelligence_migrator;
ALTER SCHEMA naeos_intelligence OWNER TO naeos_intelligence_migrator;
GRANT USAGE ON SCHEMA naeos_intelligence TO naeos_intelligence_runtime;
GRANT CREATE ON SCHEMA naeos_intelligence TO naeos_intelligence_migrator;
REVOKE CREATE ON SCHEMA naeos_intelligence FROM naeos_intelligence_runtime;

ALTER ROLE naeos_intelligence_runtime_login SET search_path TO 'pg_catalog, naeos_intelligence';
ALTER ROLE naeos_intelligence_migrator_login SET search_path TO 'pg_catalog, naeos_intelligence';
