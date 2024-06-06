# `trainmap-api`
This is a way to serve an API with my `trainmap-db` Go package (with possibility of adding a frontend webroot) using a custom config.

# Config
| Field                     | Meaning                                                                          |
| ------------------------- | -------------------------------------------------------------------------------- |
| `host_port`               | The post to host the server on                                                   |
| `sight_day_preview_count` | How many days should the observation timespan be                                 |
| `database_filepath`       | Path to the database (which can be generated with the `trainmap-loader` example) |
| `serve_frontend`          | Whether or not to serve a frontend on a custom dir                               |
| `frontend_root`           | If serving a frontend, the path to the frontend file root                        |
# Endpoints
See [endpoints.md](./endpoints.md) for a list of endpoints.

# See also
- https://github.com/rom-vtn/trainmap-db (core package)
- https://github.com/rom-vtn/trainmap-loader (the package to build a database with)
- https://github.com/rom-vtn/trainmap-site (the webroot I'm using)