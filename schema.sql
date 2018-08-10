--
-- PostgreSQL database dump
--

-- Dumped from database version 9.6.1
-- Dumped by pg_dump version 9.6.1

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SET check_function_bodies = false;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: plpgsql; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS plpgsql WITH SCHEMA pg_catalog;


--
-- Name: EXTENSION plpgsql; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION plpgsql IS 'PL/pgSQL procedural language';


--
-- Name: hstore; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS hstore WITH SCHEMA public;


--
-- Name: EXTENSION hstore; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION hstore IS 'data type for storing sets of (key, value) pairs';


--
-- Name: uuid-ossp; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS "uuid-ossp" WITH SCHEMA public;


--
-- Name: EXTENSION "uuid-ossp"; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION "uuid-ossp" IS 'generate universally unique identifiers (UUIDs)';


SET search_path = public, pg_catalog;

SET default_tablespace = '';

SET default_with_oids = false;

--
-- Name: apps; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE apps (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    name character varying(30) NOT NULL,
    created_at timestamp without time zone DEFAULT timezone('utc'::text, now()),
    repo text,
    exposure text DEFAULT 'private'::text NOT NULL,
    certs json,
    maintenance boolean DEFAULT false NOT NULL,
    deleted_at timestamp without time zone
);


--
-- Name: certificates; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE certificates (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    app_id uuid NOT NULL,
    name text,
    certificate_chain text,
    created_at timestamp without time zone DEFAULT timezone('utc'::text, now()),
    updated_at timestamp without time zone DEFAULT timezone('utc'::text, now())
);


--
-- Name: configs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE configs (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    app_id uuid NOT NULL,
    vars hstore,
    created_at timestamp without time zone DEFAULT timezone('utc'::text, now())
);


--
-- Name: domains; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE domains (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    app_id uuid NOT NULL,
    hostname text NOT NULL,
    created_at timestamp without time zone DEFAULT timezone('utc'::text, now())
);


--
-- Name: ecs_environment; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE ecs_environment (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    environment json NOT NULL
);


--
-- Name: ports; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE ports (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    port integer,
    app_id uuid,
    taken text
);


--
-- Name: releases; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE releases (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    app_id uuid NOT NULL,
    config_id uuid NOT NULL,
    slug_id uuid NOT NULL,
    version integer NOT NULL,
    description text,
    created_at timestamp without time zone DEFAULT timezone('utc'::text, now()),
    formation json NOT NULL
);


--
-- Name: scheduler_migration; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE scheduler_migration (
    app_id text NOT NULL,
    backend text NOT NULL
);


--
-- Name: schema_migrations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE schema_migrations (
    version integer NOT NULL
);


--
-- Name: slugs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE slugs (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    image text NOT NULL,
    procfile bytea NOT NULL
);


--
-- Name: stacks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE stacks (
    app_id text NOT NULL,
    stack_name text NOT NULL
);


--
-- Name: apps apps_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY apps
    ADD CONSTRAINT apps_pkey PRIMARY KEY (id);


--
-- Name: certificates certificates_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY certificates
    ADD CONSTRAINT certificates_pkey PRIMARY KEY (id);


--
-- Name: configs configs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY configs
    ADD CONSTRAINT configs_pkey PRIMARY KEY (id);


--
-- Name: domains domains_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY domains
    ADD CONSTRAINT domains_pkey PRIMARY KEY (id);


--
-- Name: ecs_environment ecs_environment_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY ecs_environment
    ADD CONSTRAINT ecs_environment_pkey PRIMARY KEY (id);


--
-- Name: ports ports_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY ports
    ADD CONSTRAINT ports_pkey PRIMARY KEY (id);


--
-- Name: releases releases_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY releases
    ADD CONSTRAINT releases_pkey PRIMARY KEY (id);


--
-- Name: schema_migrations schema_migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY schema_migrations
    ADD CONSTRAINT schema_migrations_pkey PRIMARY KEY (version);


--
-- Name: slugs slugs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY slugs
    ADD CONSTRAINT slugs_pkey PRIMARY KEY (id);


--
-- Name: index_certificates_on_app_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_certificates_on_app_id ON certificates USING btree (app_id);


--
-- Name: index_configs_on_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_configs_on_created_at ON configs USING btree (created_at);


--
-- Name: index_domains_on_app_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_domains_on_app_id ON domains USING btree (app_id);


--
-- Name: index_domains_on_hostname; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_domains_on_hostname ON domains USING btree (hostname);


--
-- Name: index_releases_on_app_id_and_version; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_releases_on_app_id_and_version ON releases USING btree (app_id, version);


--
-- Name: index_stacks_on_app_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_stacks_on_app_id ON stacks USING btree (app_id);


--
-- Name: index_stacks_on_stack_name; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_stacks_on_stack_name ON stacks USING btree (stack_name);


--
-- Name: unique_app_name; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX unique_app_name ON apps USING btree (name) WHERE (deleted_at IS NULL);


--
-- Name: certificates certificates_app_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY certificates
    ADD CONSTRAINT certificates_app_id_fkey FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE;


--
-- Name: configs configs_app_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY configs
    ADD CONSTRAINT configs_app_id_fkey FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE;


--
-- Name: domains domains_app_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY domains
    ADD CONSTRAINT domains_app_id_fkey FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE;


--
-- Name: ports ports_app_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY ports
    ADD CONSTRAINT ports_app_id_fkey FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE SET NULL;


--
-- Name: releases releases_app_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY releases
    ADD CONSTRAINT releases_app_id_fkey FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE;


--
-- Name: releases releases_config_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY releases
    ADD CONSTRAINT releases_config_id_fkey FOREIGN KEY (config_id) REFERENCES configs(id) ON DELETE CASCADE;


--
-- Name: releases releases_slug_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY releases
    ADD CONSTRAINT releases_slug_id_fkey FOREIGN KEY (slug_id) REFERENCES slugs(id) ON DELETE CASCADE;


--
-- PostgreSQL database dump complete
--

