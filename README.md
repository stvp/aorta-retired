Aorta
=====

Aorta is a sister app for Red. It will do the following:

* Hold open stable Redis connections to Redis servers
* Return INFO, CONFIG, and SLOWLOG results that are cached for any duration and
  can be refreshed on a per-request basis

Common response codes
=====================

The following response codes can be returned by any API endpoint:

* 401 - Your request was not authenticated.
* 500 - Aorta barfed. Check the server logs for details.
* 503 - Aorta could not connect to Redis. See the response body for details.
* 504 - Aorta timed out while trying to connect / communicate with Redis.

Error responses
===============

The body will be JSON formatted like so:

    {
      error: "TODO"
    }

API
===

GET /info
---------

Params:

* url -- The Redis URL
* group (optional) -- Fetch a non-default INFO group
* maxage (optional) -- If the cached INFO is older than this number of
  seconds, it will be fetched fresh

Response:

    {
      "redis_version": "2.6.0",
      "redis_git_sha1": 729d801a,
      ...
    }

Database stats, command stats, and slave stats are all broken out into nested
object:

    {
      ...
      "db": {
        "0": {
          "keys": "1045",
          "expires": "301"
        }
      },
      ...
      "cmdstat": {
        "get": {
          "calls": "44347727",
          "usec": "362625427",
          "usec_per_call": "8.18"
        },
        ...
      },
      ...
      "slave": {
        "0": {
          "ip": "127.0.0.1",
          "port": "6604",
          "state": "online",
          "offset": "29",     // Redis >= 2.8.0
          "lag": "1"          // Redis >= 2.8.0
        },
        ...
      }
    }

GET /config
-----------

Params:

* url
* maxage

GET /slowlog
------------

Params:

* url
* maxage (optional)
* count (optional) -- Number of slowlog entries to return

Response:

GET /status
-----------

Returns info about our connection to the given URL (most important is when we
last heard from the given Redis server).

Params:

* url

Response:

    {
      url: "redis://localhost:6379",
      last_activity: 1395174780,
    }

