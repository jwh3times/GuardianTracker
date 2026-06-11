-- Dev-only bootstrap for docker-compose. Tables come from api-service embedded migrations.
CREATE ROLE guardian_app LOGIN PASSWORD 'guardian_dev_password';
GRANT CONNECT ON DATABASE guardian_tracker TO guardian_app;
GRANT CREATE, USAGE ON SCHEMA public TO guardian_app;
