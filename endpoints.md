# API Endpoints
> Note: still in development, expect breaking changes and/or not up do date docs. Feel free to request more docs if reading the code doesn't give you any idea of how the responses are structured.

- `<type>` = `"trips"|"routes"|"stops"`

| Endpoint                                     | Description                                           |
| -------------------------------------------- | ----------------------------------------------------- |
| `GET /api/sights/<lat>/<lon>/<YYYY-MM-DD>`   | Get train sights starting at a given date             |
| `GET /api/sights/<lat>/<lon>`                | Gets train sights with observation starting yesterday |
| `GET /api/data/<type>/<feed_id>/<object_id>` | Gets info about the requested object                  |
| `GET /api/data/feeds`                        | Shows all feeds and their IDs                         |