CREATE DATABASE payminto;
CREATE USER payminto WITH PASSWORD 'payminto_dev';
GRANT ALL PRIVILEGES ON DATABASE payminto TO payminto;
ALTER DATABASE payminto OWNER TO payminto;
\c payminto
GRANT ALL ON SCHEMA public TO payminto;
