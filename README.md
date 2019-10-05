# pdns-grpc

Manage DNS Resource Record with PowerDNS, gRPC, and PostgreSQL.

![](https://github.com/KoyamaSohei/pdns-grpc/workflows/main/badge.svg)

## Usage

### docker-compose 

Please check [example](https://github.com/KoyamaSohei/pdns-grpc/tree/master/example)

## Environment Variables

- SOA_MNAME(required)

  The <domain-name> of the name server that was the original or primary source of data for this zone.

- SOA_RNAME(required)

  A <domain-name> which specifies the mailbox of the person responsible for this zone.

- GRPC_HOST(default = `"0.0.0.0"`)

  host which this package listening on.

- GRPC_PORT(default = `"50051"`)

  port which this package listening on.

- GPGSQL_HOST(default = `"postgres"`)

  host which this package connect to postgresql on.

- GPGSQL_USER(default = `"postgres"`)

  user which this package connect to postgresql by.

- GPGSQL_PASSWORD(default = `""`)

  password which this package connect to postgresql with.

- GPGSQL_DBNAME(default = `"postgres"`)

  database name of postgres which this package connect to.
