# bleve-facilities-poc

Local proof of concept for [RB-285](https://warbyparker.atlassian.net/browse/RB-285): time a [Bleve](https://blevesearch.com) index of retail facilities against the searches `POST /v2/facilities/search` already runs.

This module is not part of `retail-services`. It does not change the retail API or the Elasticsearch queries.

## What it measures

Search time only, for:

- a text query (`Greene`)
- a latitude/longitude query with a distance filter, ordered by distance

The stage endpoint also loads facilities from DynamoDB and runs a second search for appointment-type options. Those steps are outside this measurement.
