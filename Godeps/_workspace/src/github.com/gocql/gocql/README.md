gocql
=====

[![Build Status](https://travis-ci.org/gocql/gocql.png?branch=master)](https://travis-ci.org/gocql/gocql)
[![GoDoc](http://godoc.org/github.com/gocql/gocql?status.png)](http://godoc.org/github.com/gocql/gocql)

Package gocql implements a fast and robust Cassandra client for the
Go programming language.

Project Website: http://gocql.github.io/<br>
API documentation: http://godoc.org/github.com/gocql/gocql<br>
Discussions: https://groups.google.com/forum/#!forum/gocql

Supported Versions
------------------

The following matrix shows the versions of Go and Cassandra that are tested with the integration test suite as part of the CI build:

Go/Cassandra | 1.2.19 | 2.0.11 | 2.1.2
-------------| -------| ------| ---------
1.2  | yes | yes | yes
1.3  | yes | yes | yes

Installation
------------

    go get github.com/gocql/gocql


Features
--------

* Modern Cassandra client using the native transport
* Automatic type conversations between Cassandra and Go
  * Support for all common types including sets, lists and maps
  * Custom types can implement a `Marshaler` and `Unmarshaler` interface
  * Strict type conversations without any loss of precision
  * Built-In support for UUIDs (version 1 and 4)
* Support for logged, unlogged and counter batches
* Cluster management
  * Automatic reconnect on connection failures with exponential falloff
  * Round robin distribution of queries to different hosts
  * Round robin distribution of queries to different connections on a host
  * Each connection can execute up to 128 concurrent queries
  * Optional automatic discovery of nodes
  * Optional support for periodic node discovery via system.peers
* Iteration over paged results with configurable page size
* Support for TLS/SSL
* Optional frame compression (using snappy)
* Automatic query preparation
* Support for query tracing

Please visit the [Roadmap](https://github.com/gocql/gocql/wiki/Roadmap) page to see what is on the horizion.

Important Default Keyspace Changes
----------------------------------
gocql no longer supports executing "use <keyspace>" statements to simplfy the library. The user still has the
ability to define the default keyspace for connections but now the keyspace can only be defined before a
session is created. Queries can still access keyspaces by indicating the keyspace in the query:
```sql
SELECT * FROM example2.table;
```

Example of correct usage:
```go
	cluster := gocql.NewCluster("192.168.1.1", "192.168.1.2", "192.168.1.3")
	cluster.Keyspace = "example"
	...
	session, err := cluster.CreateSession()

```
Example of incorrect usage:
```go
	cluster := gocql.NewCluster("192.168.1.1", "192.168.1.2", "192.168.1.3")
	cluster.Keyspace = "example"
	...
	session, err := cluster.CreateSession()

	if err = session.Query("use example2").Exec(); err != nil {
		log.Fatal(err)
	}
```
This will result in an err being returned from the session.Query line as the user is trying to execute a "use"
statement. 

Example
-------

```go
/* Before you execute the program, Launch `cqlsh` and execute:
create keyspace example with replication = { 'class' : 'SimpleStrategy', 'replication_factor' : 1 };
create table example.tweet(timeline text, id UUID, text text, PRIMARY KEY(id));
create index on example.tweet(timeline);
*/
package main

import (
	"fmt"
	"log"

	"github.com/gocql/gocql"
)

func main() {
	// connect to the cluster
	cluster := gocql.NewCluster("192.168.1.1", "192.168.1.2", "192.168.1.3")
	cluster.Keyspace = "example"
	cluster.Consistency = gocql.Quorum
	session, _ := cluster.CreateSession()
	defer session.Close()

	// insert a tweet
	if err := session.Query(`INSERT INTO tweet (timeline, id, text) VALUES (?, ?, ?)`,
		"me", gocql.TimeUUID(), "hello world").Exec(); err != nil {
		log.Fatal(err)
	}

	var id gocql.UUID
	var text string

	/* Search for a specific set of records whose 'timeline' column matches
	 * the value 'me'. The secondary index that we created earlier will be
	 * used for optimizing the search */
	if err := session.Query(`SELECT id, text FROM tweet WHERE timeline = ? LIMIT 1`,
		"me").Consistency(gocql.One).Scan(&id, &text); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Tweet:", id, text)

	// list all tweets
	iter := session.Query(`SELECT id, text FROM tweet WHERE timeline = ?`, "me").Iter()
	for iter.Scan(&id, &text) {
		fmt.Println("Tweet:", id, text)
	}
	if err := iter.Close(); err != nil {
		log.Fatal(err)
	}
}
```

Data Binding
------------

There are various ways to bind application level data structures to CQL statements:

* You can write the data binding by hand, as outlined in the Tweet example. This provides you with the greatest flexibility, but it does mean that you need to keep your application code in sync with your Cassandra schema.
* You can dynamically marshal an entire query result into an `[]map[string]interface{}` using the `SliceMap()` API. This returns a slice of row maps keyed by CQL column mames. This method requires no special interaction with the gocql API, but it does require your application to be able to deal with a key value view of your data.
* As a refinement on the `SliceMap()` API you can also call `MapScan()` which returns `map[string]interface{}` instances in a row by row fashion.
* The `Bind()` API provides a client app with a low level mechanism to introspect query meta data and extract appropriate field values from application level data structures.
* Building on top of the gocql driver, [cqlr](https://github.com/relops/cqlr) adds the ability to auto-bind a CQL iterator to a struct or to bind a struct to an INSERT statement.
* Another external project that layers on top of gocql is [cqlc](http://relops.com/cqlc) which generates gocql compliant code from your Cassandra schema so that you can write type safe CQL statements in Go with a natural query syntax.

Ecosphere
---------

The following community maintained tools are known to integrate with gocql:

* [migrate](https://github.com/mattes/migrate) is a migration handling tool written in Go with Cassandra support.
* [negronicql](https://github.com/mikebthun/negronicql) is gocql middleware for Negroni.
* [cqlr](https://github.com/relops/cqlr) adds the ability to auto-bind a CQL iterator to a struct or to bind a struct to an INSERT statement.
* [cqlc](http://relops.com/cqlc) which generates gocql compliant code from your Cassandra schema so that you can write type safe CQL statements in Go with a natural query syntax.

Other Projects
--------------

* [gocqldriver](https://github.com/tux21b/gocqldriver) is the predecessor of gocql based on Go's "database/sql" package. This project isn't maintained anymore, because Cassandra wasn't a good fit for the traditional "database/sql" API. Use this package instead.

SEO
---

For some reason, when you google `golang cassandra`, this project doesn't feature very highly in the result list. But if you google `go cassandra`, then we're a bit higher up the list. So this is note to try to convince Google that Golang is an alias for Go.

License
-------

> Copyright (c) 2012-2014 The gocql Authors. All rights reserved.
> Use of this source code is governed by a BSD-style
> license that can be found in the LICENSE file.
