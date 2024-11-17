# JsonStore
Simple and not efficient http handler to store any Json data.


# HTTP handler
sample usage

CRUD single
POST /db/<key> payload:{}
GET /db/<key> -> json
GET /db -> get all keys with query paramters, like pagination / limit / sort
DELETE /db/<key>



# Storage
several storage alternatives are provided, and you can also create your own fulfilling the interface TODO

## DB using gorm
db is a simple kv implementation that uses gorm to store the jsons as string in a sql database



## Goals
 * make a general purpose kv store where i can store a random json with a key
 * make it so that also a collection can be stored in a key
 * http endpoints for crud
 * persistance using sql/gorm
 * persistance using plain text json
 * use collection name concatenated with useage to differenciate between users


## http

