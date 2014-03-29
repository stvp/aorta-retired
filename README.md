Aorta
=====

Aorta is a Redis proxy server that holds open Redis connections to any number of
servers and allows for optional command caching. It behaves like a regular Redis
server except with the addition of a few extra commands.

Commands
--------

### PROXY host port auth

Proxy all following commands to the given Redis server.

### CACHED seconds command [args...]

Return cached results for the given command. If the cache is older than
`seconds`, fresh results will be fetched, cached, and returned.

