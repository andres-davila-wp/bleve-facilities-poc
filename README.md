# bleve-facilities-poc

Local proof of concept for [RB-285](https://warbyparker.atlassian.net/browse/RB-285): time a [Bleve](https://blevesearch.com) index of retail facilities against the searches `POST /v2/facilities/search` already runs.

This module is not part of `retail-services`. It does not change the retail API or the Elasticsearch queries.

The measurement and the decision are in [docs/findings.md](docs/findings.md). The four Postman requests are in [postman/POC-Blevesearch.postman_collection.json](postman/POC-Blevesearch.postman_collection.json).

## What it measures

Search time only, for:

- a text query (`Greene`)
- a latitude/longitude query with a distance filter, ordered by distance

The stage endpoint also loads facilities from DynamoDB and runs a second search for appointment-type options. Those steps are outside this measurement.

## Run

```bash
go test .
go run . -data testdata/facilities.json
go run . -serve -data testdata/facilities-stage.json
```

`-serve` listens on `127.0.0.1:8787` and accepts the same `POST /v2/facilities/search` query string as stage. The response field `tookMs` is the search time.

`testdata/facilities.json` is four sample stores so the program runs. Those times are not comparable to stage. Replace the file with a JSON array of the real facilities, then point `-lat` and `-lon` at a real search:

```bash
go run . -data facilities.json -term Greene -lat 40.7260 -lon -73.9960 -distance 25 -unit mi
```

Each object uses `id`, `name`, `streetAddress`, `postalCode`, `locality`, `storeNumber`, `countryCode`, `type`, `isActive`, `hasLaunched`, `lat`, and `lon`. Text search matches name, street, postal code, locality, store number, country, and id. A hit must be active and launched. The distance search keeps facilities inside `-distance` and orders them nearest first.
