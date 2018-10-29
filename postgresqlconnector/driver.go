package postgresqlconnector

import (
	"bytes"
	"database/sql/driver"
	"sort"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/go-sql-driver/mysql"
	"github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

var _ driver.Driver = (*PostgreSQLLoggingDriver)(nil)

type PostgreSQLLoggingDriver struct {
	Logger *logrus.Logger
	Driver *pq.Driver
}

func FromConfigString(s string) PostgreSQLOpts {
	opts := PostgreSQLOpts{}
	for _, kv := range strings.Split(s, " ") {
		kvs := strings.Split(kv, "=")
		if len(kvs) > 1 {
			opts[kvs[0]] = kvs[1]
		}
	}
	return opts
}

type PostgreSQLOpts map[string]string

func (opts PostgreSQLOpts) String() string {
	buf := bytes.NewBuffer(nil)

	kvs := make([]string, 0)
	for k := range opts {
		kvs = append(kvs, k)
	}
	sort.Strings(kvs)

	for i, k := range kvs {
		if i > 0 {
			buf.WriteByte(' ')
		}
		buf.WriteString(k)
		buf.WriteByte('=')
		buf.WriteString(opts[k])
	}

	return buf.String()
}

func (d *PostgreSQLLoggingDriver) Open(dsn string) (driver.Conn, error) {
	conf, err := pq.ParseURL(dsn)
	if err != nil {
		panic(err)
	}
	opts := FromConfigString(conf)
	if pass, ok := opts["password"]; ok {
		opts["password"] = strings.Repeat("*", len(pass))
	}

	conn, err := d.Driver.Open(conf)
	if err != nil {
		d.Logger.Errorf("failed to open connection: %s %s", opts, err)
		return nil, err
	}
	d.Logger.Debugf(color.YellowString("connected %s", opts))
	return &loggerConn{cfg: opts, conn: conn, logger: d.Logger}, nil
}

var _ interface {
	driver.Conn
} = (*loggerConn)(nil)

type loggerConn struct {
	logger *logrus.Logger
	cfg    PostgreSQLOpts
	conn   driver.Conn
}

func (c *loggerConn) Begin() (driver.Tx, error) {
	c.logger.Debugf(color.YellowString("=========== Beginning Transaction ==========="))
	tx, err := c.conn.Begin()
	if err != nil {
		c.logger.Errorf("failed to begin transaction: %s", err)
		return nil, err
	}
	return &loggingTx{tx: tx, logger: c.logger}, nil
}

func (c *loggerConn) Close() error {
	if err := c.conn.Close(); err != nil {
		c.logger.Errorf("failed to close connection: %s", err)
		return err
	}
	return nil
}

func (c *loggerConn) Prepare(query string) (driver.Stmt, error) {
	stmt, err := c.conn.Prepare(query)
	if err != nil {
		c.logger.Errorf("failed to prepare query: %s, err: %s", query, err)
		return nil, err
	}
	return &loggerStmt{cfg: c.cfg, query: query, stmt: stmt, logger: c.logger}, nil
}

var _ driver.Stmt = (*loggerStmt)(nil)

type loggerStmt struct {
	logger *logrus.Logger
	cfg    PostgreSQLOpts
	query  string
	stmt   driver.Stmt
}

func (s *loggerStmt) Close() error {
	if err := s.stmt.Close(); err != nil {
		s.logger.Errorf("failed to close statement: %s", err)
		return err
	}
	return nil
}

var DuplicateEntryErrNumber uint16 = 1062

func startTimer() func() time.Duration {
	startTime := time.Now()
	return func() time.Duration {
		return time.Now().Sub(startTime)
	}
}

func (s *loggerStmt) Exec(args []driver.Value) (driver.Result, error) {
	cost := startTimer()

	if len(args) != 0 {
		sqlForLog, err := interpolateParams(s.query, args)
		if err != nil {
			s.logger.Warnf("failed exec %s: %s", err, color.RedString(s.query))
			return nil, err
		}
		s.query = sqlForLog
	}

	result, err := s.stmt.Exec(args)
	if err != nil {
		if mysqlErr, ok := err.(*mysql.MySQLError); !ok {
			s.logger.Errorf("failed exec %s: %s", err, color.RedString(s.query))
		} else if mysqlErr.Number == DuplicateEntryErrNumber {
			s.logger.Warnf("failed exec %s: %s", err, color.RedString(s.query))
		} else {
			s.logger.Errorf("failed exec %s: %s", err, color.RedString(s.query))
		}
		return nil, err
	}

	s.logger.WithField("cost", cost().String()).Debugf(color.YellowString(s.query))
	return result, nil
}

func (s *loggerStmt) Query(args []driver.Value) (driver.Rows, error) {
	cost := startTimer()

	if len(args) != 0 {
		sqlForLog, err := interpolateParams(s.query, args)
		if err != nil {
			if mysqlErr, ok := err.(*mysql.MySQLError); !ok {
				s.logger.Errorf("failed exec %s: %s", err, color.RedString(s.query))
			} else {
				s.logger.Warnf("failed exec %s: %s", mysqlErr, color.RedString(s.query))
			}
			return nil, err
		}
		s.query = sqlForLog
	}

	rows, err := s.stmt.Query(args)
	if err != nil {
		s.logger.Warnf("failed query %s: %s", err, color.RedString(s.query))
		return nil, err
	}

	s.logger.WithField("cost", cost().String()).Debugf(color.GreenString(s.query))
	return rows, nil
}

func (s *loggerStmt) NumInput() int {
	i := s.stmt.NumInput()
	return i
}

type loggingTx struct {
	logger *logrus.Logger
	tx     driver.Tx
}

func (tx *loggingTx) Commit() error {
	if err := tx.tx.Commit(); err != nil {
		tx.logger.Debugf("failed to commit transaction: %s", err)
		return err
	}
	tx.logger.Debugf(color.YellowString("=========== Committed Transaction ==========="))
	return nil
}

func (tx *loggingTx) Rollback() error {
	if err := tx.tx.Rollback(); err != nil {
		tx.logger.Debugf("failed to rollback transaction: %s", err)
		return err
	}
	tx.logger.Debugf("=========== Rollback Transaction ===========")
	return nil
}