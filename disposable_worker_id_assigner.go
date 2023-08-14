package uidgenerator

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"math/rand"
	"time"
)

// DisposableWorkerIdAssigner Represents an implementation of WorkerIdAssigner
// the worker id will be discarded after assigned to the UidGenerator
type DisposableWorkerIdAssigner struct {
	db        *sql.DB
	insertSql string
	selectSql string
}

func NewDisposableWorkerIdAssigner(dsn string) (*DisposableWorkerIdAssigner, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	return &DisposableWorkerIdAssigner{
		db: db,
		insertSql: `
INSERT INTO WORKER_NODE
    (HOST_NAME,PORT,TYPE,LAUNCH_DATE,MODIFIED,CREATED) 
VALUES 
    (?,?,?,?,NOW(),NOW())
`,
		selectSql: `
SELECT
		ID,
		HOST_NAME,
		PORT,
		TYPE,
		LAUNCH_DATE,
		MODIFIED,
		CREATED
FROM
		WORKER_NODE
WHERE
		HOST_NAME = ? AND PORT = ?
`,
	}, nil
}

/*
Assign worker id base on database.
If there is host name & port in the environment, we considered that the node runs in Docker container
Otherwise, the node runs on an actual machine.
*/
func (d *DisposableWorkerIdAssigner) assignWorkerId() (int64, error) {
	// build worker node entity
	workerNode := d.buildWorkerNode()
	result, err := d.db.Exec(d.insertSql, workerNode.HostName, workerNode.Port, workerNode.Type, workerNode.launchDate)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (d *DisposableWorkerIdAssigner) buildWorkerNode() *workerNode {
	workerNode := workerNode{}
	workerNode.launchDate = time.Now()
	if dockerInfo.IsDocker {
		workerNode.Type = container
		workerNode.HostName = dockerInfo.Host
		workerNode.Port = dockerInfo.Port
	} else {
		workerNode.Type = actual
		workerNode.HostName = netInfo.LocalAddress
		source := rand.NewSource(time.Now().UnixMilli())
		r := rand.New(source)
		workerNode.Port = fmt.Sprintf("%d-%d", time.Now().UnixMilli(), r.Intn(100000))
	}
	return &workerNode
}
