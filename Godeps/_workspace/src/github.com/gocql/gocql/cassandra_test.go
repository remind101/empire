// +build all integration

package gocql

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"math"
	"math/big"
	"net"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode"

	"speter.net/go/exp/math/dec/inf"
)

var (
	flagCluster    = flag.String("cluster", "127.0.0.1", "a comma-separated list of host:port tuples")
	flagProto      = flag.Int("proto", 2, "protcol version")
	flagCQL        = flag.String("cql", "3.0.0", "CQL version")
	flagRF         = flag.Int("rf", 1, "replication factor for test keyspace")
	clusterSize    = flag.Int("clusterSize", 1, "the expected size of the cluster")
	flagRetry      = flag.Int("retries", 5, "number of times to retry queries")
	flagAutoWait   = flag.Duration("autowait", 1000*time.Millisecond, "time to wait for autodiscovery to fill the hosts poll")
	flagRunSslTest = flag.Bool("runssl", false, "Set to true to run ssl test")
	clusterHosts   []string
)

func init() {
	flag.Parse()
	clusterHosts = strings.Split(*flagCluster, ",")
	log.SetFlags(log.Lshortfile | log.LstdFlags)
}

func addSslOptions(cluster *ClusterConfig) *ClusterConfig {
	if *flagRunSslTest {
		cluster.SslOpts = &SslOptions{
			CertPath:               "testdata/pki/gocql.crt",
			KeyPath:                "testdata/pki/gocql.key",
			CaPath:                 "testdata/pki/ca.crt",
			EnableHostVerification: false,
		}
	}
	return cluster
}

var initOnce sync.Once

func createTable(s *Session, table string) error {
	err := s.Query(table).Consistency(All).Exec()
	if *clusterSize > 1 {
		// wait for table definition to propogate
		time.Sleep(250 * time.Millisecond)
	}
	return err
}

func createCluster() *ClusterConfig {
	cluster := NewCluster(clusterHosts...)
	cluster.ProtoVersion = *flagProto
	cluster.CQLVersion = *flagCQL
	cluster.Timeout = 5 * time.Second
	cluster.Consistency = Quorum
	if *flagRetry > 0 {
		cluster.RetryPolicy = &SimpleRetryPolicy{NumRetries: *flagRetry}
	}
	cluster = addSslOptions(cluster)
	return cluster
}

func createKeyspace(tb testing.TB, cluster *ClusterConfig, keyspace string) {
	session, err := cluster.CreateSession()
	if err != nil {
		tb.Fatal("createSession:", err)
	}
	if err = session.Query(`DROP KEYSPACE ` + keyspace).Exec(); err != nil {
		tb.Log("drop keyspace:", err)
	}
	if err := session.Query(fmt.Sprintf(`CREATE KEYSPACE %s
	WITH replication = {
		'class' : 'SimpleStrategy',
		'replication_factor' : %d
	}`, keyspace, *flagRF)).Consistency(All).Exec(); err != nil {
		tb.Fatalf("error creating keyspace %s: %v", keyspace, err)
	}
	tb.Logf("Created keyspace %s", keyspace)
	session.Close()
}

func createSession(tb testing.TB) *Session {
	cluster := createCluster()

	// Drop and re-create the keyspace once. Different tests should use their own
	// individual tables, but can assume that the table does not exist before.
	initOnce.Do(func() {
		createKeyspace(tb, cluster, "gocql_test")
	})

	cluster.Keyspace = "gocql_test"
	session, err := cluster.CreateSession()
	if err != nil {
		tb.Fatal("createSession:", err)
	}

	return session
}

//TestRingDiscovery makes sure that you can autodiscover other cluster members when you seed a cluster config with just one node
func TestRingDiscovery(t *testing.T) {

	cluster := NewCluster(clusterHosts[0])
	cluster.ProtoVersion = *flagProto
	cluster.CQLVersion = *flagCQL
	cluster.Timeout = 5 * time.Second
	cluster.Consistency = Quorum
	if *flagRetry > 0 {
		cluster.RetryPolicy = &SimpleRetryPolicy{NumRetries: *flagRetry}
	}
	cluster.DiscoverHosts = true
	cluster = addSslOptions(cluster)
	session, err := cluster.CreateSession()
	if err != nil {
		t.Errorf("got error connecting to the cluster %v", err)
	}

	if *clusterSize > 1 {
		// wait for autodiscovery to update the pool with the list of known hosts
		time.Sleep(*flagAutoWait)
	}

	size := len(session.Pool.(*SimplePool).connPool)

	if *clusterSize != size {
		t.Logf("WARN: Expected a cluster size of %d, but actual size was %d", *clusterSize, size)
	}

	session.Close()
}

func TestEmptyHosts(t *testing.T) {
	cluster := NewCluster()
	cluster = addSslOptions(cluster)
	if session, err := cluster.CreateSession(); err == nil {
		session.Close()
		t.Error("expected err, got nil")
	}
}

//TestUseStatementError checks to make sure the correct error is returned when the user tries to execute a use statement.
func TestUseStatementError(t *testing.T) {
	session := createSession(t)
	defer session.Close()

	if err := session.Query("USE gocql_test").Exec(); err != nil {
		if err != ErrUseStmt {
			t.Error("expected ErrUseStmt, got " + err.Error())
		}
	} else {
		t.Error("expected err, got nil.")
	}
}

//TestInvalidKeyspace checks that an invalid keyspace will return promptly and without a flood of connections
func TestInvalidKeyspace(t *testing.T) {
	cluster := NewCluster(clusterHosts...)
	cluster.ProtoVersion = *flagProto
	cluster.CQLVersion = *flagCQL
	cluster.Keyspace = "invalidKeyspace"
	cluster = addSslOptions(cluster)
	session, err := cluster.CreateSession()
	if err != nil {
		if err != ErrNoConnectionsStarted {
			t.Errorf("Expected ErrNoConnections but got %v", err)
		}
	} else {
		session.Close() //Clean up the session
		t.Error("expected err, got nil.")
	}
}

func TestTracing(t *testing.T) {
	session := createSession(t)
	defer session.Close()

	if err := createTable(session, `CREATE TABLE trace (id int primary key)`); err != nil {
		t.Fatal("create:", err)
	}

	buf := &bytes.Buffer{}
	trace := NewTraceWriter(session, buf)

	if err := session.Query(`INSERT INTO trace (id) VALUES (?)`, 42).Trace(trace).Exec(); err != nil {
		t.Error("insert:", err)
	} else if buf.Len() == 0 {
		t.Error("insert: failed to obtain any tracing")
	}
	buf.Reset()

	var value int
	if err := session.Query(`SELECT id FROM trace WHERE id = ?`, 42).Trace(trace).Scan(&value); err != nil {
		t.Error("select:", err)
	} else if value != 42 {
		t.Errorf("value: expected %d, got %d", 42, value)
	} else if buf.Len() == 0 {
		t.Error("select: failed to obtain any tracing")
	}
}

func TestPaging(t *testing.T) {
	if *flagProto == 1 {
		t.Skip("Paging not supported. Please use Cassandra >= 2.0")
	}

	session := createSession(t)
	defer session.Close()

	if err := createTable(session, "CREATE TABLE paging (id int primary key)"); err != nil {
		t.Fatal("create table:", err)
	}
	for i := 0; i < 100; i++ {
		if err := session.Query("INSERT INTO paging (id) VALUES (?)", i).Exec(); err != nil {
			t.Fatal("insert:", err)
		}
	}

	iter := session.Query("SELECT id FROM paging").PageSize(10).Iter()
	var id int
	count := 0
	for iter.Scan(&id) {
		count++
	}
	if err := iter.Close(); err != nil {
		t.Fatal("close:", err)
	}
	if count != 100 {
		t.Fatalf("expected %d, got %d", 100, count)
	}
}

func TestCAS(t *testing.T) {
	if *flagProto == 1 {
		t.Skip("lightweight transactions not supported. Please use Cassandra >= 2.0")
	}

	session := createSession(t)
	defer session.Close()

	if err := createTable(session, `CREATE TABLE cas_table (
			title         varchar,
			revid   	  timeuuid,
			last_modified timestamp,
			PRIMARY KEY (title, revid)
		)`); err != nil {
		t.Fatal("create:", err)
	}

	title, revid, modified := "baz", TimeUUID(), time.Now()
	var titleCAS string
	var revidCAS UUID
	var modifiedCAS time.Time

	if applied, err := session.Query(`INSERT INTO cas_table (title, revid, last_modified)
		VALUES (?, ?, ?) IF NOT EXISTS`,
		title, revid, modified).ScanCAS(&titleCAS, &revidCAS, &modifiedCAS); err != nil {
		t.Fatal("insert:", err)
	} else if !applied {
		t.Fatal("insert should have been applied")
	}

	if applied, err := session.Query(`INSERT INTO cas_table (title, revid, last_modified)
		VALUES (?, ?, ?) IF NOT EXISTS`,
		title, revid, modified).ScanCAS(&titleCAS, &revidCAS, &modifiedCAS); err != nil {
		t.Fatal("insert:", err)
	} else if applied {
		t.Fatal("insert should not have been applied")
	} else if title != titleCAS || revid != revidCAS {
		t.Fatalf("expected %s/%v/%v but got %s/%v/%v", title, revid, modified, titleCAS, revidCAS, modifiedCAS)
	}

	tenSecondsLater := modified.Add(10 * time.Second)

	if applied, err := session.Query(`DELETE FROM cas_table WHERE title = ? and revid = ? IF last_modified = ?`,
		title, revid, tenSecondsLater).ScanCAS(&modifiedCAS); err != nil {
		t.Fatal("delete:", err)
	} else if applied {
		t.Fatal("delete should have not been applied")
	}

	if modifiedCAS.Unix() != tenSecondsLater.Add(-10*time.Second).Unix() {
		t.Fatalf("Was expecting modified CAS to be %v; but was one second later", modifiedCAS.UTC())
	}

	if _, err := session.Query(`DELETE FROM cas_table WHERE title = ? and revid = ? IF last_modified = ?`,
		title, revid, tenSecondsLater).ScanCAS(); err.Error() != "count mismatch" {
		t.Fatalf("delete: was expecting count mismatch error but got %s", err)
	}

	if applied, err := session.Query(`DELETE FROM cas_table WHERE title = ? and revid = ? IF last_modified = ?`,
		title, revid, modified).ScanCAS(&modifiedCAS); err != nil {
		t.Fatal("delete:", err)
	} else if !applied {
		t.Fatal("delete should have been applied")
	}
}

func TestMapScanCAS(t *testing.T) {
	if *flagProto == 1 {
		t.Skip("lightweight transactions not supported. Please use Cassandra >= 2.0")
	}

	session := createSession(t)
	defer session.Close()

	if err := createTable(session, `CREATE TABLE cas_table2 (
			title         varchar,
			revid   	  timeuuid,
			last_modified timestamp,
			deleted boolean,
			PRIMARY KEY (title, revid)
		)`); err != nil {
		t.Fatal("create:", err)
	}

	title, revid, modified, deleted := "baz", TimeUUID(), time.Now(), false
	mapCAS := map[string]interface{}{}

	if applied, err := session.Query(`INSERT INTO cas_table2 (title, revid, last_modified, deleted)
		VALUES (?, ?, ?, ?) IF NOT EXISTS`,
		title, revid, modified, deleted).MapScanCAS(mapCAS); err != nil {
		t.Fatal("insert:", err)
	} else if !applied {
		t.Fatal("insert should have been applied")
	}

	mapCAS = map[string]interface{}{}
	if applied, err := session.Query(`INSERT INTO cas_table2 (title, revid, last_modified, deleted)
		VALUES (?, ?, ?, ?) IF NOT EXISTS`,
		title, revid, modified, deleted).MapScanCAS(mapCAS); err != nil {
		t.Fatal("insert:", err)
	} else if applied {
		t.Fatal("insert should not have been applied")
	} else if title != mapCAS["title"] || revid != mapCAS["revid"] || deleted != mapCAS["deleted"] {
		t.Fatalf("expected %s/%v/%v/%v but got %s/%v/%v%v", title, revid, modified, false, mapCAS["title"], mapCAS["revid"], mapCAS["last_modified"], mapCAS["deleted"])
	}

}

func TestBatch(t *testing.T) {
	if *flagProto == 1 {
		t.Skip("atomic batches not supported. Please use Cassandra >= 2.0")
	}

	session := createSession(t)
	defer session.Close()

	if err := createTable(session, `CREATE TABLE batch_table (id int primary key)`); err != nil {
		t.Fatal("create table:", err)
	}

	batch := NewBatch(LoggedBatch)
	for i := 0; i < 100; i++ {
		batch.Query(`INSERT INTO batch_table (id) VALUES (?)`, i)
	}
	if err := session.ExecuteBatch(batch); err != nil {
		t.Fatal("execute batch:", err)
	}

	count := 0
	if err := session.Query(`SELECT COUNT(*) FROM batch_table`).Scan(&count); err != nil {
		t.Fatal("select count:", err)
	} else if count != 100 {
		t.Fatalf("count: expected %d, got %d\n", 100, count)
	}
}

// TestBatchLimit tests gocql to make sure batch operations larger than the maximum
// statement limit are not submitted to a cassandra node.
func TestBatchLimit(t *testing.T) {
	if *flagProto == 1 {
		t.Skip("atomic batches not supported. Please use Cassandra >= 2.0")
	}
	session := createSession(t)
	defer session.Close()

	if err := createTable(session, `CREATE TABLE batch_table2 (id int primary key)`); err != nil {
		t.Fatal("create table:", err)
	}

	batch := NewBatch(LoggedBatch)
	for i := 0; i < 65537; i++ {
		batch.Query(`INSERT INTO batch_table2 (id) VALUES (?)`, i)
	}
	if err := session.ExecuteBatch(batch); err != ErrTooManyStmts {
		t.Fatal("gocql attempted to execute a batch larger than the support limit of statements.")
	}

}

// TestTooManyQueryArgs tests to make sure the library correctly handles the application level bug
// whereby too many query arguments are passed to a query
func TestTooManyQueryArgs(t *testing.T) {
	if *flagProto == 1 {
		t.Skip("atomic batches not supported. Please use Cassandra >= 2.0")
	}
	session := createSession(t)
	defer session.Close()

	if err := createTable(session, `CREATE TABLE too_many_query_args (id int primary key, value int)`); err != nil {
		t.Fatal("create table:", err)
	}

	_, err := session.Query(`SELECT * FROM too_many_query_args WHERE id = ?`, 1, 2).Iter().SliceMap()

	if err == nil {
		t.Fatal("'`SELECT * FROM too_many_query_args WHERE id = ?`, 1, 2' should return an ErrQueryArgLength")
	}

	if err != ErrQueryArgLength {
		t.Fatalf("'`SELECT * FROM too_many_query_args WHERE id = ?`, 1, 2' should return an ErrQueryArgLength, but returned: %s", err)
	}

	batch := session.NewBatch(UnloggedBatch)
	batch.Query("INSERT INTO too_many_query_args (id, value) VALUES (?, ?)", 1, 2, 3)
	err = session.ExecuteBatch(batch)

	if err == nil {
		t.Fatal("'`INSERT INTO too_many_query_args (id, value) VALUES (?, ?)`, 1, 2, 3' should return an ErrQueryArgLength")
	}

	if err != ErrQueryArgLength {
		t.Fatalf("'INSERT INTO too_many_query_args (id, value) VALUES (?, ?)`, 1, 2, 3' should return an ErrQueryArgLength, but returned: %s", err)
	}

}

// TestNotEnoughQueryArgs tests to make sure the library correctly handles the application level bug
// whereby not enough query arguments are passed to a query
func TestNotEnoughQueryArgs(t *testing.T) {
	if *flagProto == 1 {
		t.Skip("atomic batches not supported. Please use Cassandra >= 2.0")
	}
	session := createSession(t)
	defer session.Close()

	if err := createTable(session, `CREATE TABLE not_enough_query_args (id int, cluster int, value int, primary key (id, cluster))`); err != nil {
		t.Fatal("create table:", err)
	}

	_, err := session.Query(`SELECT * FROM not_enough_query_args WHERE id = ? and cluster = ?`, 1).Iter().SliceMap()

	if err == nil {
		t.Fatal("'`SELECT * FROM not_enough_query_args WHERE id = ? and cluster = ?`, 1' should return an ErrQueryArgLength")
	}

	if err != ErrQueryArgLength {
		t.Fatalf("'`SELECT * FROM too_few_query_args WHERE id = ? and cluster = ?`, 1' should return an ErrQueryArgLength, but returned: %s", err)
	}

	batch := session.NewBatch(UnloggedBatch)
	batch.Query("INSERT INTO not_enough_query_args (id, cluster, value) VALUES (?, ?, ?)", 1, 2)
	err = session.ExecuteBatch(batch)

	if err == nil {
		t.Fatal("'`INSERT INTO not_enough_query_args (id, cluster, value) VALUES (?, ?, ?)`, 1, 2' should return an ErrQueryArgLength")
	}

	if err != ErrQueryArgLength {
		t.Fatalf("'INSERT INTO not_enough_query_args (id, cluster, value) VALUES (?, ?, ?)`, 1, 2' should return an ErrQueryArgLength, but returned: %s", err)
	}
}

// TestCreateSessionTimeout tests to make sure the CreateSession function timeouts out correctly
// and prevents an infinite loop of connection retries.
func TestCreateSessionTimeout(t *testing.T) {
	go func() {
		<-time.After(2 * time.Second)
		t.Fatal("no startup timeout")
	}()
	c := NewCluster("127.0.0.1:1")
	c = addSslOptions(c)
	_, err := c.CreateSession()

	if err == nil {
		t.Fatal("expected ErrNoConnectionsStarted, but no error was returned.")
	}
	if err != ErrNoConnectionsStarted {
		t.Fatalf("expected ErrNoConnectionsStarted, but received %v", err)
	}
}

type FullName struct {
	FirstName string
	LastName  string
}

func (n FullName) MarshalCQL(info *TypeInfo) ([]byte, error) {
	return []byte(n.FirstName + " " + n.LastName), nil
}
func (n *FullName) UnmarshalCQL(info *TypeInfo, data []byte) error {
	t := strings.SplitN(string(data), " ", 2)
	n.FirstName, n.LastName = t[0], t[1]
	return nil
}

func TestMapScanWithRefMap(t *testing.T) {
	session := createSession(t)
	defer session.Close()
	if err := createTable(session, `CREATE TABLE scan_map_ref_table (
			testtext       text PRIMARY KEY,
			testfullname   text,
			testint        int,
		)`); err != nil {
		t.Fatal("create table:", err)
	}
	m := make(map[string]interface{})
	m["testtext"] = "testtext"
	m["testfullname"] = FullName{"John", "Doe"}
	m["testint"] = 100

	if err := session.Query(`INSERT INTO scan_map_ref_table (testtext, testfullname, testint) values (?,?,?)`, m["testtext"], m["testfullname"], m["testint"]).Exec(); err != nil {
		t.Fatal("insert:", err)
	}

	var testText string
	var testFullName FullName
	ret := map[string]interface{}{
		"testtext":     &testText,
		"testfullname": &testFullName,
		// testint is not set here.
	}
	iter := session.Query(`SELECT * FROM scan_map_ref_table`).Iter()
	if ok := iter.MapScan(ret); !ok {
		t.Fatal("select:", iter.Close())
	} else {
		if ret["testtext"] != "testtext" {
			t.Fatal("returned testtext did not match")
		}
		f := ret["testfullname"].(FullName)
		if f.FirstName != "John" || f.LastName != "Doe" {
			t.Fatal("returned testfullname did not match")
		}
		if ret["testint"] != 100 {
			t.Fatal("returned testinit did not match")
		}
	}

}

func TestSliceMap(t *testing.T) {
	session := createSession(t)
	defer session.Close()
	if err := createTable(session, `CREATE TABLE slice_map_table (
			testuuid       timeuuid PRIMARY KEY,
			testtimestamp  timestamp,
			testvarchar    varchar,
			testbigint     bigint,
			testblob       blob,
			testbool       boolean,
			testfloat      float,
			testdouble     double,
			testint        int,
			testdecimal    decimal,
			testlist       list<text>,
			testset        set<int>,
			testmap        map<varchar, varchar>,
			testvarint     varint,
			testinet			 inet
		)`); err != nil {
		t.Fatal("create table:", err)
	}
	m := make(map[string]interface{})

	bigInt := new(big.Int)
	if _, ok := bigInt.SetString("830169365738487321165427203929228", 10); !ok {
		t.Fatal("Failed setting bigint by string")
	}

	m["testuuid"] = TimeUUID()
	m["testvarchar"] = "Test VarChar"
	m["testbigint"] = time.Now().Unix()
	m["testtimestamp"] = time.Now().Truncate(time.Millisecond).UTC()
	m["testblob"] = []byte("test blob")
	m["testbool"] = true
	m["testfloat"] = float32(4.564)
	m["testdouble"] = float64(4.815162342)
	m["testint"] = 2343
	m["testdecimal"] = inf.NewDec(100, 0)
	m["testlist"] = []string{"quux", "foo", "bar", "baz", "quux"}
	m["testset"] = []int{1, 2, 3, 4, 5, 6, 7, 8, 9}
	m["testmap"] = map[string]string{"field1": "val1", "field2": "val2", "field3": "val3"}
	m["testvarint"] = bigInt
	m["testinet"] = "213.212.2.19"
	sliceMap := []map[string]interface{}{m}
	if err := session.Query(`INSERT INTO slice_map_table (testuuid, testtimestamp, testvarchar, testbigint, testblob, testbool, testfloat, testdouble, testint, testdecimal, testlist, testset, testmap, testvarint, testinet) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		m["testuuid"], m["testtimestamp"], m["testvarchar"], m["testbigint"], m["testblob"], m["testbool"], m["testfloat"], m["testdouble"], m["testint"], m["testdecimal"], m["testlist"], m["testset"], m["testmap"], m["testvarint"], m["testinet"]).Exec(); err != nil {
		t.Fatal("insert:", err)
	}
	if returned, retErr := session.Query(`SELECT * FROM slice_map_table`).Iter().SliceMap(); retErr != nil {
		t.Fatal("select:", retErr)
	} else {
		matchSliceMap(t, sliceMap, returned[0])
	}

	// Test for Iter.MapScan()
	{
		testMap := make(map[string]interface{})
		if !session.Query(`SELECT * FROM slice_map_table`).Iter().MapScan(testMap) {
			t.Fatal("MapScan failed to work with one row")
		}
		matchSliceMap(t, sliceMap, testMap)
	}

	// Test for Query.MapScan()
	{
		testMap := make(map[string]interface{})
		if session.Query(`SELECT * FROM slice_map_table`).MapScan(testMap) != nil {
			t.Fatal("MapScan failed to work with one row")
		}
		matchSliceMap(t, sliceMap, testMap)
	}
}
func matchSliceMap(t *testing.T, sliceMap []map[string]interface{}, testMap map[string]interface{}) {
	if sliceMap[0]["testuuid"] != testMap["testuuid"] {
		t.Fatal("returned testuuid did not match")
	}
	if sliceMap[0]["testtimestamp"] != testMap["testtimestamp"] {
		t.Fatal("returned testtimestamp did not match")
	}
	if sliceMap[0]["testvarchar"] != testMap["testvarchar"] {
		t.Fatal("returned testvarchar did not match")
	}
	if sliceMap[0]["testbigint"] != testMap["testbigint"] {
		t.Fatal("returned testbigint did not match")
	}
	if !reflect.DeepEqual(sliceMap[0]["testblob"], testMap["testblob"]) {
		t.Fatal("returned testblob did not match")
	}
	if sliceMap[0]["testbool"] != testMap["testbool"] {
		t.Fatal("returned testbool did not match")
	}
	if sliceMap[0]["testfloat"] != testMap["testfloat"] {
		t.Fatal("returned testfloat did not match")
	}
	if sliceMap[0]["testdouble"] != testMap["testdouble"] {
		t.Fatal("returned testdouble did not match")
	}
	if sliceMap[0]["testinet"] != testMap["testinet"] {
		t.Fatal("returned testinet did not match")
	}

	expectedDecimal := sliceMap[0]["testdecimal"].(*inf.Dec)
	returnedDecimal := testMap["testdecimal"].(*inf.Dec)

	if expectedDecimal.Cmp(returnedDecimal) != 0 {
		t.Fatal("returned testdecimal did not match")
	}

	if !reflect.DeepEqual(sliceMap[0]["testlist"], testMap["testlist"]) {
		t.Fatal("returned testlist did not match")
	}
	if !reflect.DeepEqual(sliceMap[0]["testset"], testMap["testset"]) {
		t.Fatal("returned testset did not match")
	}
	if !reflect.DeepEqual(sliceMap[0]["testmap"], testMap["testmap"]) {
		t.Fatal("returned testmap did not match")
	}
	if sliceMap[0]["testint"] != testMap["testint"] {
		t.Fatal("returned testint did not match")
	}
}

func TestScanWithNilArguments(t *testing.T) {
	session := createSession(t)
	defer session.Close()

	if err := createTable(session, `CREATE TABLE scan_with_nil_arguments (
			foo   varchar,
			bar   int,
			PRIMARY KEY (foo, bar)
	)`); err != nil {
		t.Fatal("create:", err)
	}
	for i := 1; i <= 20; i++ {
		if err := session.Query("INSERT INTO scan_with_nil_arguments (foo, bar) VALUES (?, ?)",
			"squares", i*i).Exec(); err != nil {
			t.Fatal("insert:", err)
		}
	}

	iter := session.Query("SELECT * FROM scan_with_nil_arguments WHERE foo = ?", "squares").Iter()
	var n int
	count := 0
	for iter.Scan(nil, &n) {
		count += n
	}
	if err := iter.Close(); err != nil {
		t.Fatal("close:", err)
	}
	if count != 2870 {
		t.Fatalf("expected %d, got %d", 2870, count)
	}
}

func TestScanCASWithNilArguments(t *testing.T) {
	if *flagProto == 1 {
		t.Skip("lightweight transactions not supported. Please use Cassandra >= 2.0")
	}

	session := createSession(t)
	defer session.Close()

	if err := createTable(session, `CREATE TABLE scan_cas_with_nil_arguments (
		foo   varchar,
		bar   varchar,
		PRIMARY KEY (foo, bar)
	)`); err != nil {
		t.Fatal("create:", err)
	}

	foo := "baz"
	var cas string

	if applied, err := session.Query(`INSERT INTO scan_cas_with_nil_arguments (foo, bar)
		VALUES (?, ?) IF NOT EXISTS`,
		foo, foo).ScanCAS(nil, nil); err != nil {
		t.Fatal("insert:", err)
	} else if !applied {
		t.Fatal("insert should have been applied")
	}

	if applied, err := session.Query(`INSERT INTO scan_cas_with_nil_arguments (foo, bar)
		VALUES (?, ?) IF NOT EXISTS`,
		foo, foo).ScanCAS(&cas, nil); err != nil {
		t.Fatal("insert:", err)
	} else if applied {
		t.Fatal("insert should not have been applied")
	} else if foo != cas {
		t.Fatalf("expected %v but got %v", foo, cas)
	}

	if applied, err := session.Query(`INSERT INTO scan_cas_with_nil_arguments (foo, bar)
		VALUES (?, ?) IF NOT EXISTS`,
		foo, foo).ScanCAS(nil, &cas); err != nil {
		t.Fatal("insert:", err)
	} else if applied {
		t.Fatal("insert should not have been applied")
	} else if foo != cas {
		t.Fatalf("expected %v but got %v", foo, cas)
	}
}

func TestRebindQueryInfo(t *testing.T) {
	session := createSession(t)
	defer session.Close()

	if err := createTable(session, "CREATE TABLE rebind_query (id int, value text, PRIMARY KEY (id))"); err != nil {
		t.Fatalf("failed to create table with error '%v'", err)
	}

	if err := session.Query("INSERT INTO rebind_query (id, value) VALUES (?, ?)", 23, "quux").Exec(); err != nil {
		t.Fatalf("insert into rebind_query failed, err '%v'", err)
	}

	if err := session.Query("INSERT INTO rebind_query (id, value) VALUES (?, ?)", 24, "w00t").Exec(); err != nil {
		t.Fatalf("insert into rebind_query failed, err '%v'", err)
	}

	q := session.Query("SELECT value FROM rebind_query WHERE ID = ?")
	q.Bind(23)

	iter := q.Iter()
	var value string
	for iter.Scan(&value) {
	}

	if value != "quux" {
		t.Fatalf("expected %v but got %v", "quux", value)
	}

	q.Bind(24)
	iter = q.Iter()

	for iter.Scan(&value) {
	}

	if value != "w00t" {
		t.Fatalf("expected %v but got %v", "quux", value)
	}
}

//TestStaticQueryInfo makes sure that the application can manually bind query parameters using the simplest possible static binding strategy
func TestStaticQueryInfo(t *testing.T) {
	session := createSession(t)
	defer session.Close()

	if err := createTable(session, "CREATE TABLE static_query_info (id int, value text, PRIMARY KEY (id))"); err != nil {
		t.Fatalf("failed to create table with error '%v'", err)
	}

	if err := session.Query("INSERT INTO static_query_info (id, value) VALUES (?, ?)", 113, "foo").Exec(); err != nil {
		t.Fatalf("insert into static_query_info failed, err '%v'", err)
	}

	autobinder := func(q *QueryInfo) ([]interface{}, error) {
		values := make([]interface{}, 1)
		values[0] = 113
		return values, nil
	}

	qry := session.Bind("SELECT id, value FROM static_query_info WHERE id = ?", autobinder)

	if err := qry.Exec(); err != nil {
		t.Fatalf("expose query info failed, error '%v'", err)
	}

	iter := qry.Iter()

	var id int
	var value string

	iter.Scan(&id, &value)

	if err := iter.Close(); err != nil {
		t.Fatalf("query with exposed info failed, err '%v'", err)
	}

	if value != "foo" {
		t.Fatalf("Expected value %s, but got %s", "foo", value)
	}

}

type ClusteredKeyValue struct {
	Id      int
	Cluster int
	Value   string
}

func (kv *ClusteredKeyValue) Bind(q *QueryInfo) ([]interface{}, error) {
	values := make([]interface{}, len(q.Args))

	for i, info := range q.Args {
		fieldName := upcaseInitial(info.Name)
		value := reflect.ValueOf(kv)
		field := reflect.Indirect(value).FieldByName(fieldName)
		values[i] = field.Addr().Interface()
	}

	return values, nil
}

func upcaseInitial(str string) string {
	for i, v := range str {
		return string(unicode.ToUpper(v)) + str[i+1:]
	}
	return ""
}

//TestBoundQueryInfo makes sure that the application can manually bind query parameters using the query meta data supplied at runtime
func TestBoundQueryInfo(t *testing.T) {

	session := createSession(t)
	defer session.Close()

	if err := createTable(session, "CREATE TABLE clustered_query_info (id int, cluster int, value text, PRIMARY KEY (id, cluster))"); err != nil {
		t.Fatalf("failed to create table with error '%v'", err)
	}

	write := &ClusteredKeyValue{Id: 200, Cluster: 300, Value: "baz"}

	insert := session.Bind("INSERT INTO clustered_query_info (id, cluster, value) VALUES (?, ?,?)", write.Bind)

	if err := insert.Exec(); err != nil {
		t.Fatalf("insert into clustered_query_info failed, err '%v'", err)
	}

	read := &ClusteredKeyValue{Id: 200, Cluster: 300}

	qry := session.Bind("SELECT id, cluster, value FROM clustered_query_info WHERE id = ? and cluster = ?", read.Bind)

	iter := qry.Iter()

	var id, cluster int
	var value string

	iter.Scan(&id, &cluster, &value)

	if err := iter.Close(); err != nil {
		t.Fatalf("query with clustered_query_info info failed, err '%v'", err)
	}

	if value != "baz" {
		t.Fatalf("Expected value %s, but got %s", "baz", value)
	}

}

//TestBatchQueryInfo makes sure that the application can manually bind query parameters when executing in a batch
func TestBatchQueryInfo(t *testing.T) {

	if *flagProto == 1 {
		t.Skip("atomic batches not supported. Please use Cassandra >= 2.0")
	}

	session := createSession(t)
	defer session.Close()

	if err := createTable(session, "CREATE TABLE batch_query_info (id int, cluster int, value text, PRIMARY KEY (id, cluster))"); err != nil {
		t.Fatalf("failed to create table with error '%v'", err)
	}

	write := func(q *QueryInfo) ([]interface{}, error) {
		values := make([]interface{}, 3)
		values[0] = 4000
		values[1] = 5000
		values[2] = "bar"
		return values, nil
	}

	batch := session.NewBatch(LoggedBatch)
	batch.Bind("INSERT INTO batch_query_info (id, cluster, value) VALUES (?, ?,?)", write)

	if err := session.ExecuteBatch(batch); err != nil {
		t.Fatalf("batch insert into batch_query_info failed, err '%v'", err)
	}

	read := func(q *QueryInfo) ([]interface{}, error) {
		values := make([]interface{}, 2)
		values[0] = 4000
		values[1] = 5000
		return values, nil
	}

	qry := session.Bind("SELECT id, cluster, value FROM batch_query_info WHERE id = ? and cluster = ?", read)

	iter := qry.Iter()

	var id, cluster int
	var value string

	iter.Scan(&id, &cluster, &value)

	if err := iter.Close(); err != nil {
		t.Fatalf("query with batch_query_info info failed, err '%v'", err)
	}

	if value != "bar" {
		t.Fatalf("Expected value %s, but got %s", "bar", value)
	}
}

func injectInvalidPreparedStatement(t *testing.T, session *Session, table string) (string, *Conn) {
	if err := createTable(session, `CREATE TABLE `+table+` (
			foo   varchar,
			bar   int,
			PRIMARY KEY (foo, bar)
	)`); err != nil {
		t.Fatal("create:", err)
	}
	stmt := "INSERT INTO " + table + " (foo, bar) VALUES (?, 7)"
	conn := session.Pool.Pick(nil)
	flight := new(inflightPrepare)
	stmtsLRU.Lock()
	stmtsLRU.lru.Add(conn.addr+stmt, flight)
	stmtsLRU.Unlock()
	flight.info = &QueryInfo{
		Id: []byte{'f', 'o', 'o', 'b', 'a', 'r'},
		Args: []ColumnInfo{ColumnInfo{
			Keyspace: "gocql_test",
			Table:    table,
			Name:     "foo",
			TypeInfo: &TypeInfo{
				Type: TypeVarchar,
			},
		}},
	}
	return stmt, conn
}

func TestMissingSchemaPrepare(t *testing.T) {
	s := createSession(t)
	conn := s.Pool.Pick(nil)
	defer s.Close()

	insertQry := &Query{stmt: "INSERT INTO invalidschemaprep (val) VALUES (?)", values: []interface{}{5}, cons: s.cons,
		session: s, pageSize: s.pageSize, trace: s.trace,
		prefetch: s.prefetch, rt: s.cfg.RetryPolicy}

	if err := conn.executeQuery(insertQry).err; err == nil {
		t.Fatal("expected error, but got nil.")
	}

	if err := createTable(s, "CREATE TABLE invalidschemaprep (val int, PRIMARY KEY (val))"); err != nil {
		t.Fatal("create table:", err)
	}

	if err := conn.executeQuery(insertQry).err; err != nil {
		t.Fatal(err) // unconfigured columnfamily
	}
}

func TestReprepareStatement(t *testing.T) {
	session := createSession(t)
	defer session.Close()
	stmt, conn := injectInvalidPreparedStatement(t, session, "test_reprepare_statement")
	query := session.Query(stmt, "bar")
	if err := conn.executeQuery(query).Close(); err != nil {
		t.Fatalf("Failed to execute query for reprepare statement: %v", err)
	}
}

func TestReprepareBatch(t *testing.T) {
	if *flagProto == 1 {
		t.Skip("atomic batches not supported. Please use Cassandra >= 2.0")
	}
	session := createSession(t)
	defer session.Close()
	stmt, conn := injectInvalidPreparedStatement(t, session, "test_reprepare_statement_batch")
	batch := session.NewBatch(UnloggedBatch)
	batch.Query(stmt, "bar")
	if err := conn.executeBatch(batch); err != nil {
		t.Fatalf("Failed to execute query for reprepare statement: %v", err)
	}

}

func TestQueryInfo(t *testing.T) {
	session := createSession(t)
	defer session.Close()

	conn := session.Pool.Pick(nil)
	info, err := conn.prepareStatement("SELECT release_version, host_id FROM system.local WHERE key = ?", nil)

	if err != nil {
		t.Fatalf("Failed to execute query for preparing statement: %v", err)
	}

	if len(info.Args) != 1 {
		t.Fatalf("Was not expecting meta data for %d query arguments, but got %d\n", 1, len(info.Args))
	}

	if *flagProto > 1 {
		if len(info.Rval) != 2 {
			t.Fatalf("Was not expecting meta data for %d result columns, but got %d\n", 2, len(info.Rval))
		}
	}
}

//TestPreparedCacheEviction will make sure that the cache size is maintained
func TestPreparedCacheEviction(t *testing.T) {
	session := createSession(t)
	defer session.Close()
	stmtsLRU.Lock()
	stmtsLRU.Max(4)
	stmtsLRU.Unlock()

	if err := createTable(session, "CREATE TABLE prepcachetest (id int,mod int,PRIMARY KEY (id))"); err != nil {
		t.Fatalf("failed to create table with error '%v'", err)
	}
	//Fill the table
	for i := 0; i < 2; i++ {
		if err := session.Query("INSERT INTO prepcachetest (id,mod) VALUES (?, ?)", i, 10000%(i+1)).Exec(); err != nil {
			t.Fatalf("insert into prepcachetest failed, err '%v'", err)
		}
	}
	//Populate the prepared statement cache with select statements
	var id, mod int
	for i := 0; i < 2; i++ {
		err := session.Query("SELECT id,mod FROM prepcachetest WHERE id = "+strconv.FormatInt(int64(i), 10)).Scan(&id, &mod)
		if err != nil {
			t.Fatalf("select from prepcachetest failed, error '%v'", err)
		}
	}

	//generate an update statement to test they are prepared
	err := session.Query("UPDATE prepcachetest SET mod = ? WHERE id = ?", 1, 11).Exec()
	if err != nil {
		t.Fatalf("update prepcachetest failed, error '%v'", err)
	}

	//generate a delete statement to test they are prepared
	err = session.Query("DELETE FROM prepcachetest WHERE id = ?", 1).Exec()
	if err != nil {
		t.Fatalf("delete from prepcachetest failed, error '%v'", err)
	}

	//generate an insert statement to test they are prepared
	err = session.Query("INSERT INTO prepcachetest (id,mod) VALUES (?, ?)", 3, 11).Exec()
	if err != nil {
		t.Fatalf("insert into prepcachetest failed, error '%v'", err)
	}

	//Make sure the cache size is maintained
	if stmtsLRU.lru.Len() != stmtsLRU.lru.MaxEntries {
		t.Fatalf("expected cache size of %v, got %v", stmtsLRU.lru.MaxEntries, stmtsLRU.lru.Len())
	}

	//Walk through all the configured hosts and test cache retention and eviction
	var selFound, insFound, updFound, delFound, selEvict bool
	for i := range session.cfg.Hosts {
		_, ok := stmtsLRU.lru.Get(session.cfg.Hosts[i] + ":9042gocql_testSELECT id,mod FROM prepcachetest WHERE id = 1")
		selFound = selFound || ok

		_, ok = stmtsLRU.lru.Get(session.cfg.Hosts[i] + ":9042gocql_testINSERT INTO prepcachetest (id,mod) VALUES (?, ?)")
		insFound = insFound || ok

		_, ok = stmtsLRU.lru.Get(session.cfg.Hosts[i] + ":9042gocql_testUPDATE prepcachetest SET mod = ? WHERE id = ?")
		updFound = updFound || ok

		_, ok = stmtsLRU.lru.Get(session.cfg.Hosts[i] + ":9042gocql_testDELETE FROM prepcachetest WHERE id = ?")
		delFound = delFound || ok

		_, ok = stmtsLRU.lru.Get(session.cfg.Hosts[i] + ":9042gocql_testSELECT id,mod FROM prepcachetest WHERE id = 0")
		selEvict = selEvict || !ok

	}
	if !selEvict {
		t.Fatalf("expected first select statement to be purged, but statement was found in the cache.")
	}
	if !selFound {
		t.Fatalf("expected second select statement to be cached, but statement was purged or not prepared.")
	}
	if !insFound {
		t.Fatalf("expected insert statement to be cached, but statement was purged or not prepared.")
	}
	if !updFound {
		t.Fatalf("expected update statement to be cached, but statement was purged or not prepared.")
	}
	if !delFound {
		t.Error("expected delete statement to be cached, but statement was purged or not prepared.")
	}
}

func TestPreparedCacheKey(t *testing.T) {
	session := createSession(t)
	defer session.Close()

	// create a second keyspace
	cluster2 := createCluster()
	createKeyspace(t, cluster2, "gocql_test2")
	cluster2.Keyspace = "gocql_test2"
	session2, err := cluster2.CreateSession()
	if err != nil {
		t.Fatal("create session:", err)
	}
	defer session2.Close()

	// both keyspaces have a table named "test_stmt_cache_key"
	if err := createTable(session, "CREATE TABLE test_stmt_cache_key (id varchar primary key, field varchar)"); err != nil {
		t.Fatal("create table:", err)
	}
	if err := createTable(session2, "CREATE TABLE test_stmt_cache_key (id varchar primary key, field varchar)"); err != nil {
		t.Fatal("create table:", err)
	}

	// both tables have a single row with the same partition key but different column value
	if err = session.Query(`INSERT INTO test_stmt_cache_key (id, field) VALUES (?, ?)`, "key", "one").Exec(); err != nil {
		t.Fatal("insert:", err)
	}
	if err = session2.Query(`INSERT INTO test_stmt_cache_key (id, field) VALUES (?, ?)`, "key", "two").Exec(); err != nil {
		t.Fatal("insert:", err)
	}

	// should be able to see different values in each keyspace
	var value string
	if err = session.Query("SELECT field FROM test_stmt_cache_key WHERE id = ?", "key").Scan(&value); err != nil {
		t.Fatal("select:", err)
	}
	if value != "one" {
		t.Errorf("Expected one, got %s", value)
	}

	if err = session2.Query("SELECT field FROM test_stmt_cache_key WHERE id = ?", "key").Scan(&value); err != nil {
		t.Fatal("select:", err)
	}
	if value != "two" {
		t.Errorf("Expected two, got %s", value)
	}
}

//TestMarshalFloat64Ptr tests to see that a pointer to a float64 is marshalled correctly.
func TestMarshalFloat64Ptr(t *testing.T) {
	session := createSession(t)
	defer session.Close()

	if err := createTable(session, "CREATE TABLE float_test (id double, test double, primary key (id))"); err != nil {
		t.Fatal("create table:", err)
	}
	testNum := float64(7500)
	if err := session.Query(`INSERT INTO float_test (id,test) VALUES (?,?)`, float64(7500.00), &testNum).Exec(); err != nil {
		t.Fatal("insert float64:", err)
	}
}

//TestMarshalInet tests to see that a pointer to a float64 is marshalled correctly.
func TestMarshalInet(t *testing.T) {
	session := createSession(t)
	defer session.Close()

	if err := createTable(session, "CREATE TABLE inet_test (ip inet, name text, primary key (ip))"); err != nil {
		t.Fatal("create table:", err)
	}
	stringIp := "123.34.45.56"
	if err := session.Query(`INSERT INTO inet_test (ip,name) VALUES (?,?)`, stringIp, "Test IP 1").Exec(); err != nil {
		t.Fatal("insert string inet:", err)
	}
	var stringResult string
	if err := session.Query("SELECT ip FROM inet_test").Scan(&stringResult); err != nil {
		t.Fatalf("select for string from inet_test 1 failed: %v", err)
	}
	if stringResult != stringIp {
		t.Errorf("Expected %s, was %s", stringIp, stringResult)
	}

	var ipResult net.IP
	if err := session.Query("SELECT ip FROM inet_test").Scan(&ipResult); err != nil {
		t.Fatalf("select for net.IP from inet_test 1 failed: %v", err)
	}
	if ipResult.String() != stringIp {
		t.Errorf("Expected %s, was %s", stringIp, ipResult.String())
	}

	if err := session.Query(`DELETE FROM inet_test WHERE ip = ?`, stringIp).Exec(); err != nil {
		t.Fatal("delete inet table:", err)
	}

	netIp := net.ParseIP("222.43.54.65")
	if err := session.Query(`INSERT INTO inet_test (ip,name) VALUES (?,?)`, netIp, "Test IP 2").Exec(); err != nil {
		t.Fatal("insert netIp inet:", err)
	}

	if err := session.Query("SELECT ip FROM inet_test").Scan(&stringResult); err != nil {
		t.Fatalf("select for string from inet_test 2 failed: %v", err)
	}
	if stringResult != netIp.String() {
		t.Errorf("Expected %s, was %s", netIp.String(), stringResult)
	}
	if err := session.Query("SELECT ip FROM inet_test").Scan(&ipResult); err != nil {
		t.Fatalf("select for net.IP from inet_test 2 failed: %v", err)
	}
	if ipResult.String() != netIp.String() {
		t.Errorf("Expected %s, was %s", netIp.String(), ipResult.String())
	}

}

func TestVarint(t *testing.T) {
	session := createSession(t)
	defer session.Close()

	if err := createTable(session, "CREATE TABLE varint_test (id varchar, test varint, test2 varint, primary key (id))"); err != nil {
		t.Fatalf("failed to create table with error '%v'", err)
	}

	if err := session.Query(`INSERT INTO varint_test (id, test) VALUES (?, ?)`, "id", 0).Exec(); err != nil {
		t.Fatalf("insert varint: %v", err)
	}

	var result int
	if err := session.Query("SELECT test FROM varint_test").Scan(&result); err != nil {
		t.Fatalf("select from varint_test failed: %v", err)
	}

	if result != 0 {
		t.Errorf("Expected 0, was %d", result)
	}

	if err := session.Query(`INSERT INTO varint_test (id, test) VALUES (?, ?)`, "id", -1).Exec(); err != nil {
		t.Fatalf("insert varint: %v", err)
	}

	if err := session.Query("SELECT test FROM varint_test").Scan(&result); err != nil {
		t.Fatalf("select from varint_test failed: %v", err)
	}

	if result != -1 {
		t.Errorf("Expected -1, was %d", result)
	}

	if err := session.Query(`INSERT INTO varint_test (id, test) VALUES (?, ?)`, "id", int64(math.MaxInt32)+1).Exec(); err != nil {
		t.Fatalf("insert varint: %v", err)
	}

	var result64 int64
	if err := session.Query("SELECT test FROM varint_test").Scan(&result64); err != nil {
		t.Fatalf("select from varint_test failed: %v", err)
	}

	if result64 != int64(math.MaxInt32)+1 {
		t.Errorf("Expected %d, was %d", int64(math.MaxInt32)+1, result64)
	}

	biggie := new(big.Int)
	biggie.SetString("36893488147419103232", 10) // > 2**64
	if err := session.Query(`INSERT INTO varint_test (id, test) VALUES (?, ?)`, "id", biggie).Exec(); err != nil {
		t.Fatalf("insert varint: %v", err)
	}

	resultBig := new(big.Int)
	if err := session.Query("SELECT test FROM varint_test").Scan(resultBig); err != nil {
		t.Fatalf("select from varint_test failed: %v", err)
	}

	if resultBig.String() != biggie.String() {
		t.Errorf("Expected %s, was %s", biggie.String(), resultBig.String())
	}

	err := session.Query("SELECT test FROM varint_test").Scan(&result64)
	if err == nil || strings.Index(err.Error(), "out of range") == -1 {
		t.Errorf("expected out of range error since value is too big for int64")
	}

	// value not set in cassandra, leave bind variable empty
	resultBig = new(big.Int)
	if err := session.Query("SELECT test2 FROM varint_test").Scan(resultBig); err != nil {
		t.Fatalf("select from varint_test failed: %v", err)
	}

	if resultBig.Int64() != 0 {
		t.Errorf("Expected %s, was %s", biggie.String(), resultBig.String())
	}

	// can use double pointer to explicitly detect value is not set in cassandra
	if err := session.Query("SELECT test2 FROM varint_test").Scan(&resultBig); err != nil {
		t.Fatalf("select from varint_test failed: %v", err)
	}

	if resultBig != nil {
		t.Errorf("Expected %v, was %v", nil, *resultBig)
	}
}

//TestQueryStats confirms that the stats are returning valid data. Accuracy may be questionable.
func TestQueryStats(t *testing.T) {
	session := createSession(t)
	defer session.Close()
	qry := session.Query("SELECT * FROM system.peers")
	if err := qry.Exec(); err != nil {
		t.Fatalf("query failed. %v", err)
	} else {
		if qry.Attempts() < 1 {
			t.Fatal("expected at least 1 attempt, but got 0")
		}
		if qry.Latency() <= 0 {
			t.Fatalf("expected latency to be greater than 0, but got %v instead.", qry.Latency())
		}
	}
}

//TestBatchStats confirms that the stats are returning valid data. Accuracy may be questionable.
func TestBatchStats(t *testing.T) {
	if *flagProto == 1 {
		t.Skip("atomic batches not supported. Please use Cassandra >= 2.0")
	}
	session := createSession(t)
	defer session.Close()

	if err := createTable(session, "CREATE TABLE batchStats (id int, PRIMARY KEY (id))"); err != nil {
		t.Fatalf("failed to create table with error '%v'", err)
	}

	b := session.NewBatch(LoggedBatch)
	b.Query("INSERT INTO batchStats (id) VALUES (?)", 1)
	b.Query("INSERT INTO batchStats (id) VALUES (?)", 2)

	if err := session.ExecuteBatch(b); err != nil {
		t.Fatalf("query failed. %v", err)
	} else {
		if b.Attempts() < 1 {
			t.Fatal("expected at least 1 attempt, but got 0")
		}
		if b.Latency() <= 0 {
			t.Fatalf("expected latency to be greater than 0, but got %v instead.", b.Latency())
		}
	}
}

//TestNilInQuery tests to see that a nil value passed to a query is handled by Cassandra
//TODO validate the nil value by reading back the nil. Need to fix Unmarshalling.
func TestNilInQuery(t *testing.T) {
	session := createSession(t)
	defer session.Close()

	if err := createTable(session, "CREATE TABLE testNilInsert (id int, count int, PRIMARY KEY (id))"); err != nil {
		t.Fatalf("failed to create table with error '%v'", err)
	}
	if err := session.Query("INSERT INTO testNilInsert (id,count) VALUES (?,?)", 1, nil).Exec(); err != nil {
		t.Fatalf("failed to insert with err: %v", err)
	}

	var id int

	if err := session.Query("SELECT id FROM testNilInsert").Scan(&id); err != nil {
		t.Fatalf("failed to select with err: %v", err)
	} else if id != 1 {
		t.Fatalf("expected id to be 1, got %v", id)
	}
}

// Don't initialize time.Time bind variable if cassandra timestamp column is empty
func TestEmptyTimestamp(t *testing.T) {
	session := createSession(t)
	defer session.Close()

	if err := createTable(session, "CREATE TABLE test_empty_timestamp (id int, time timestamp, num int, PRIMARY KEY (id))"); err != nil {
		t.Fatalf("failed to create table with error '%v'", err)
	}

	if err := session.Query("INSERT INTO test_empty_timestamp (id, num) VALUES (?,?)", 1, 561).Exec(); err != nil {
		t.Fatalf("failed to insert with err: %v", err)
	}

	var timeVal time.Time

	if err := session.Query("SELECT time FROM test_empty_timestamp where id = ?", 1).Scan(&timeVal); err != nil {
		t.Fatalf("failed to select with err: %v", err)
	}

	if !timeVal.IsZero() {
		t.Errorf("time.Time bind variable should still be empty (was %s)", timeVal)
	}
}
