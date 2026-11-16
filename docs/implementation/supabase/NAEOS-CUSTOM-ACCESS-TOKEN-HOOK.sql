-- NAEOS IAM-08: Supabase Custom Access Token Hook
-- Grants only allowlisted NAEOS capabilities from administrator-managed
-- auth.users.raw_app_meta_data (surfaced as claims.app_metadata).
-- Never source permissions from raw_user_meta_data / JWT user_metadata.
--
-- Configure this function in Supabase Dashboard:
-- Authentication > Hooks > Custom Access Token (Postgres function).

create or replace function public.naeos_custom_access_token_hook(event jsonb)
returns jsonb
language plpgsql
stable
as $$
declare
  claims jsonb;
  requested_scopes jsonb;
  allowed_scopes jsonb;
begin
  claims := coalesce(event->'claims', '{}'::jsonb);
  requested_scopes := claims #> '{app_metadata,naeos_scopes}';

  if jsonb_typeof(requested_scopes) = 'array' then
    select coalesce(jsonb_agg(scope), '[]'::jsonb)
      into allowed_scopes
      from (
        select distinct value as scope
        from jsonb_array_elements_text(requested_scopes)
        where value in ('intelligence:read', 'intelligence:write')
      ) as allowlisted;
  else
    allowed_scopes := '[]'::jsonb;
  end if;

  -- Dedicated claim consumed by NAEOS Intelligence's verified JWT parser.
  claims := jsonb_set(claims, '{naeos_scopes}', allowed_scopes, true);
  return jsonb_set(event, '{claims}', claims, true);
end;
$$;

grant usage on schema public to supabase_auth_admin;
grant execute on function public.naeos_custom_access_token_hook(jsonb)
  to supabase_auth_admin;
revoke execute on function public.naeos_custom_access_token_hook(jsonb)
  from authenticated, anon, public;

-- Hook execution privileges are separate from assigning the hook in Dashboard.
-- No table grants are needed: permissions are carried in admin-managed app_metadata.
