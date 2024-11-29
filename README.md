# JsonStore

jsonStore is a simple (and probably not so performant) to store arbitrary data as Json strings.
It is intended for quick POCs, or use-cases where you can afford to store whole data struts and you don't need
features like search or indexing, e.g. store user settings.


## JsonStorer Interface
The JsonStorer interface defines the contract for a storage system that handles JSON data within collections.
It provides methods for CRUD operations and pagination:
* Set: Adds or updates a single item in a specified collection using a unique key and JSON-encoded value.
* Get: Retrieves the item by its key from a specified collection as raw JSON value.
* Delete: Removes an item from the collection by its key, returning whether the deletion was successful.
* List: Retrieves a paginated list of items from a collection, along with the total count of items.

This package contains several implementations of the interface

## FileStore Implementation
The FileStore implementation is JSON file based storage with optional in-memory operation.
It supports concurrency through sync.RWMutex and is configurable via flags for human-readable JSON formatting,
manual flushing.

usage:

```
file := "path/to/jsonFile.json"
// use jsonstore.InMemoryDb to not write to a file

// you can provide optional flags to the constructor
flags := []FileStoreFlag{
    ManualFlush, // if set, data will not be flushed to file but requires manually to call Flush()
    MinimizedJson, // writes minimized json insted of human readable
}
store, err := jsonstore.NewFileStore(file)

// use the interface acions:
err := storeSet(ctx, "my-collection", "item1",json.RawMessage(`{"name":"test-item"}`))
			
```


## DbStore Implementation

The DbStore is a simple key-value database persisted implementation that uses GORM as abstraction 
to implement the JsonStorer interface.

usage:

```
db := getGormDB // return a *gorm.DB 
store, err := jsonstore.NewDbStore(db)

// use the interface acions:
err := storeSet(ctx, "my-collection", "item1",json.RawMessage(`{"name":"test-item"}`))
	
```

# HTTP

## jsonstore.HttpStorer
HttpStorer extends the JsonStorer interface mapping the interface methods to http requests,
this allows to easily create http handlers that are able to store data into a JsonStorer (see Handler below)

this is the signature of HttpStorer:

```
Set(w http.ResponseWriter, r *http.Request, collection, key string)
Get(w http.ResponseWriter, r *http.Request, collection, key string) 
List(w http.ResponseWriter, r *http.Request, collection string) 
Delete(w http.ResponseWriter, r *http.Request, collection, key string)
```

## jsonstore.Handler

The Handler provides sample but usable implementation an HTTP interface to interact with a JsonStorer.
It implements the standard HTTP methods (GET, POST, DELETE) to manage JSON data,

### Initialize the handler

```
store := getMyStoreImpl() // initilalize the JsonStore

// setup the handler
handler := jsonstore.Handler{
    HttpStorer: jsonstore.HttpStorer{Storer: store},
    Collection: "my-collection-name",
}
mux := http.NewServeMux()
mux.Handle("/some/path/collection", &handler) // bind the handler to the path
```

### Create/Update
Handler POST request to store  or updates a JSON document with the specified key in the collection.
The request body must contain the JSON data to be stored.

```
POST '{"foo":"bar"}' /some/path/collection/{key}
201
```

### Get Single
Retrieves a document by key from the specified collection.

```
GET /some/path/collection/{key}
200 '{"foo":"bar"}'
```
### Get paginated list

```
GET /some/path/collection/?limit=1&page=2
200 '{
"items":[{"foo":"bar"}],
"total":3,
"page":2,
"limit":1
}'
	
```
### Delete

Delete a document by key from the specified collection.

```
DELETE /some/path/collection/{key}
200
```

## TODO

* verify that any struct can be stored and restored similar to jstore


