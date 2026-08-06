{application, dhtcrawler, [
  {description, "A DHT crawler to index magnet hash to torrent"},
  {vsn, "0.1.0"},
  {registered, [dht_crawler_sup]},
  {applications, [kernel, stdlib, crypto, public_key, ssl, inets, bson, mongodb]},
  {mod, {crawler_app, []}},
  {env, []}
]}.
