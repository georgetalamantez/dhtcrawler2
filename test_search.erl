application:start(bson),
application:start(mongodb),
{ok, Conn} = mongo_connection:start_link({localhost, 27018}),
mongo:do(unsafe, master, Conn, dhtcrawler, fun() ->
    Ret = mongo:command({text, <<"hashes">>, search, <<"test">>, limit, 50}),
    io:format("Result: ~p~n", [Ret])
end),
halt().
